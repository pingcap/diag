// Copyright 2021 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"text/template"
	"time"

	"github.com/fatih/color"
	jsoniter "github.com/json-iterator/go"
	"github.com/pingcap/errors"
	"github.com/pingcap/tiup/components/playground/instance"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/environment"
	tiupexec "github.com/pingcap/tiup/pkg/exec"
	"github.com/pingcap/tiup/pkg/localdata"
	"github.com/pingcap/tiup/pkg/repository"
	"github.com/pingcap/tiup/pkg/tui/progress"
	"github.com/pingcap/tiup/pkg/utils"
	tiuputil "github.com/pingcap/tiup/pkg/utils"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

// RebuildOptions are arguments needed for the rebuild job
type RebuildOptions struct {
	Local       bool // rebuild the system on localhost
	Host        string
	Port        int
	User        string
	Passwd      string
	DBName      string
	Cluster     string // cluster name
	Session     string // collector session ID
	File        string
	Chunk       int
	Concurrency int // max parallel jobs allowed
}

func RunLocal(dumpDir string, opt *RebuildOptions) error {
	dataDir := os.Getenv(localdata.EnvNameInstanceDataDir)
	if dataDir == "" {
		return fmt.Errorf("cannot read environment variable %s", localdata.EnvNameInstanceDataDir)
	}

	fmt.Println("Start bootstrapping a monitoring system on localhost and rebuilding the dashboards.")

	// read clsuter name
	body, err := os.ReadFile(path.Join(dumpDir, fileNameClusterJSON))
	if err != nil {
		return err
	}
	clusterJSON := map[string]interface{}{}
	err = json.Unmarshal(body, &clusterJSON)
	if err != nil {
		return err
	}
	clsName := clusterJSON["cluster_name"].(string)

	// read cluster version
	metaFile, err := os.ReadFile(filepath.Join(dumpDir, "meta.yaml"))
	if err != nil {
		return err
	}
	var meta spec.ClusterMeta
	if err := yaml.Unmarshal(metaFile, &meta); err != nil {
		return err
	}
	clsVer := meta.Version

	env, err := environment.InitEnv(repository.Options{})
	if err != nil {
		return err
	}
	environment.SetGlobalEnv(env)

	var booted uint32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := newRebuilder()
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)

		sig := (<-sc).(syscall.Signal)
		atomic.StoreInt32(&p.lastSig, int32(sig))
		fmt.Println("Received signal: ", sig)

		// if bootCluster is not done we just cancel context to make it
		// clean up and return ASAP and exit directly after timeout.
		// Note now bootCluster can not learn the context is done and return quickly now
		// like while it's downloading component.
		if atomic.LoadUint32(&booted) == 0 {
			cancel()
			time.AfterFunc(time.Second, func() {
				os.Exit(0)
			})
			return
		}

		go p.terminate(sig)
		// If user try double ctrl+c, force quit
		sig = (<-sc).(syscall.Signal)
		atomic.StoreInt32(&p.lastSig, int32(syscall.SIGKILL))
		if sig == syscall.SIGINT {
			p.terminate(syscall.SIGKILL)
			os.Exit(128 + int(sig))
		}
	}()

	bootErr := p.boot(ctx, opt, dataDir, clsVer, clsName)
	if bootErr != nil {
		// always kill all process started and wait before quit.
		atomic.StoreInt32(&p.lastSig, int32(syscall.SIGKILL))
		fmt.Printf("error bootstrap, exiting: %s\n", bootErr)
		p.terminate(syscall.SIGKILL)
		_ = p.wait()
		return errors.Annotate(bootErr, "Bootstrapping failed")
	}

	atomic.StoreUint32(&booted, 1)

	wg := sync.WaitGroup{}
	timeoutOpt := utils.RetryOption{
		Attempts: 200,
		Delay:    time.Millisecond * 300,
		Timeout:  time.Second * 60,
	}

	var loadErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		mb := progress.NewMultiBar("Starting monitor components")
		bars := make(map[string]*progress.MultiBarItem)
		for comp, ins := range p.Proc {
			bars[comp] = mb.AddBar(fmt.Sprintf(" - Setting up %s (%s)", comp, ins.addr()))
		}
		mb.StartRenderLoop()

		for comp, ins := range p.Proc {
			if err := utils.Retry(func() error {
				if ins.ready() {
					bars[comp].UpdateDisplay(&progress.DisplayProps{
						Prefix: fmt.Sprintf(" - Set up %s (%s)", comp, ins.addr()),
						Mode:   progress.ModeDone,
					})
					return nil
				}
				bars[comp].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf(" - Setting up %s (%s)", comp, ins.addr()),
					Suffix: "waiting process to be ready",
				})
				return fmt.Errorf("process of %s not ready yet", comp)
			}, timeoutOpt); err != nil {
				bars[comp].UpdateDisplay(&progress.DisplayProps{
					Prefix: fmt.Sprintf(" - Setting up %s (%s)", comp, ins.addr()),
					Suffix: err.Error(),
					Mode:   progress.ModeError,
				})
				loadErr = errors.Annotatef(err, "failed to start %s", comp)
				mb.StopRenderLoop()
				return
			}
		}
		mb.StopRenderLoop()

		loadStart := time.Now()
		loadErr = LoadMetrics(ctx, dumpDir, opt)
		fmt.Printf("Load completed in %.2f seconds\n", time.Since(loadStart).Seconds())

		// print addresses
		fmt.Println("Components are started and listening:")
		for comp, ins := range p.Proc {
			fmt.Printf("%s: %s\n", comp, color.CyanString(ins.addr()))
		}
	}()
	if loadErr != nil {
		return loadErr
	}

	var waitErr error
	go func() {
		defer wg.Done()
		waitErr = p.wait()
	}()

	wg.Wait()

	return waitErr
}

type component interface {
	start(ctx context.Context) error
	wait() error
	ready() bool
	pid() int
	getCmd() *exec.Cmd
	addr() string
}

// rebuilder is the monitoring services to run
type rebuilder struct {
	Proc    map[string]component
	walker  errgroup.Group
	lastSig int32 // latest signal recieved
	env     *environment.Environment
}

func newRebuilder() *rebuilder {
	return &rebuilder{
		Proc: make(map[string]component),
		env:  environment.GlobalEnv(),
	}
}

func (b *rebuilder) boot(ctx context.Context, opt *RebuildOptions, dataDir, clsVer, clsName string) error {
	// prepare influxdb
	if err := installIfMissing("influxdb", ""); err != nil { // latest version
		return err
	}
	var influxAddr string
	if insInflux, err := newInfluxdb(opt.Host, "", dataDir); err == nil {
		b.Proc["influxdb"] = insInflux
		influxAddr = insInflux.addr()
		opt.Port = insInflux.HTTPPort
	} else {
		return err
	}

	// prepare prometheus
	var promAddr string
	if err := installIfMissing("prometheus", clsVer); err != nil {
		return err
	}
	if insProm, err := newPrometheus(opt.Host, clsVer, dataDir, influxAddr); err == nil {
		b.Proc["prometheus"] = insProm
		promAddr = insProm.addr()
	} else {
		return err
	}

	// prepare grafana
	if err := installIfMissing("grafana", clsVer); err != nil {
		return err
	}
	grafanaDir := filepath.Join(dataDir, "grafana")
	installPath, err := b.env.Profile().ComponentInstalledPath("grafana", tiuputil.Version(clsVer))
	if err != nil {
		return err
	}
	cmd := exec.Command("cp", "-r", installPath, grafanaDir)
	err = cmd.Run()
	if err != nil {
		return errors.AddStack(err)
	}

	dashboardDir := filepath.Join(grafanaDir, "dashboards")
	err = os.MkdirAll(dashboardDir, 0755)
	if err != nil {
		return errors.AddStack(err)
	}
	// mv {grafanaDir}/*.json {grafanaDir}/dashboards/
	err = filepath.Walk(grafanaDir, func(path string, info os.FileInfo, err error) error {
		// skip scan sub directory
		if info.IsDir() && path != grafanaDir {
			return filepath.SkipDir
		}

		if strings.HasSuffix(info.Name(), ".json") {
			return os.Rename(path, filepath.Join(grafanaDir, "dashboards", info.Name()))
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = replaceDatasource(dashboardDir, clsName)
	if err != nil {
		return err
	}
	if insGrafana, err := newGrafana(opt.Host, clsVer, dataDir, clsName, promAddr); err == nil {
		b.Proc["grafana"] = insGrafana
	} else {
		return err
	}

	for comp, ins := range b.Proc {
		i := ins
		c := comp
		b.walker.Go(func() error {
			if err := i.start(ctx); err != nil {
				defer b.terminate(syscall.SIGKILL)
				return err
			}
			fmt.Printf("%s started: %s\n", c, i.addr())
			err := i.wait()
			if err != nil && atomic.LoadInt32(&b.lastSig) == 0 {
				fmt.Printf("process quit: %s: %s\n", c, err)
			}
			return err
		})
	}

	return nil
}

func (b *rebuilder) terminate(sig syscall.Signal) error {
	kill := func(pid int, wait func() error) {
		if sig != syscall.SIGINT {
			_ = syscall.Kill(pid, sig)
		}

		timer := time.AfterFunc(time.Second*10, func() {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		})

		_ = wait()
		timer.Stop()
	}

	for comp, inst := range b.Proc {
		pid := inst.pid()
		if pid == 0 { // the process does not exist
			fmt.Printf("Component %s not started, skip.", comp)
			continue
		}

		if sig == syscall.SIGKILL {
			fmt.Printf("Force %s(%d) to quit...\n", comp, pid)
		} else if atomic.LoadInt32(&b.lastSig) == int32(sig) { // In case of double ctr+c
			fmt.Printf("Wait %s(%d) to quit...\n", comp, pid)
		}

		kill(pid, inst.wait)
	}
	return nil
}

func (b *rebuilder) wait() error {
	err := b.walker.Wait()
	if err != nil && atomic.LoadInt32(&b.lastSig) == 0 {
		return err
	}
	return nil
}

type influxdb struct {
	Host     string
	BindPort int
	HTTPPort int
	Dir      string
	version  string
	cmd      *exec.Cmd

	waitErr  error
	waitOnce sync.Once
}

func (i *influxdb) start(ctx context.Context) error {
	args := []string{
		"-config", filepath.Join(i.Dir, "influxdb.conf"),
	}

	env := environment.GlobalEnv()
	os.Setenv("INFLUXD_CONFIG_PATH", i.Dir)
	params := &tiupexec.PrepareCommandParams{
		Ctx:         ctx,
		Component:   "influxdb",
		Version:     tiuputil.Version(i.version),
		InstanceDir: i.Dir,
		WD:          i.Dir,
		Args:        args,
		SysProcAttr: instance.SysProcAttr,
		Env:         env,
	}
	cmd, err := tiupexec.PrepareCommand(params)
	if err != nil {
		return err
	}
	cmd.Stdout = nil
	cmd.Stderr = nil

	i.cmd = cmd

	return i.cmd.Start()
}

func (i *influxdb) ready() bool {
	url := fmt.Sprintf("http://%s:%d/health", i.Host, i.HTTPPort)
	body, err := utils.NewHTTPClient(time.Second*2, &tls.Config{}).Get(context.TODO(), url)
	if err != nil {
		//fmt.Println("still waiting for influxdb to start...")
		return false
	}
	var status map[string]interface{}
	if err := jsoniter.Unmarshal(body, &status); err != nil {
		fmt.Printf("still waiting for influxdb to start: %s\n", err)
		return false
	}
	if r, ok := status["status"].(string); !ok || r != "pass" {
		//fmt.Println("still waiting for influxdb to start...")
		return false
	}
	return true
}

func (i *influxdb) wait() error {
	i.waitOnce.Do(func() {
		i.waitErr = i.cmd.Wait()
	})

	return i.waitErr
}

func (i *influxdb) getCmd() *exec.Cmd { return i.cmd }

func (i *influxdb) pid() int {
	if i.cmd != nil && i.cmd.Process != nil {
		return i.cmd.Process.Pid
	}
	return 0
}

func (i *influxdb) addr() string {
	return fmt.Sprintf("%s:%d", i.Host, i.HTTPPort)
}

// the cmd is not started after return
func newInfluxdb(host, version, dir string) (*influxdb, error) {
	dir = filepath.Join(dir, "influxdb")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.AddStack(err)
	}

	bindPort, err := tiuputil.GetFreePort(host, 8088)
	if err != nil {
		return nil, err
	}
	httpPort, err := tiuputil.GetFreePort(host, 8086)
	if err != nil {
		return nil, err
	}

	i := new(influxdb)
	i.Host = host
	i.Dir = dir
	i.BindPort = bindPort
	i.HTTPPort = httpPort
	i.version = version

	const influxCfg = `
bind-address = "{{.Host}}:{{.BindPort}}"
[meta]
	dir = "{{.Dir}}/meta"
[data]
	dir = "{{.Dir}}/data"
	wal-dir = "{{.Dir}}/wal"
	series-id-set-cache-size = 100
[coordinator]
[retention]
[shard-precreation]
[monitor]
[http]
	bind-address = "{{.Host}}:{{.HTTPPort}}"
[logging]
[subscriber]
[[graphite]]
[[collectd]]
[[opentsdb]]
[[udp]]
[continuous_queries]
[tls]
`

	tmpl, err := template.New("influxdb.conf").Parse(influxCfg)
	if err != nil {
		return nil, err
	}
	content := bytes.NewBufferString("")
	if err := tmpl.Execute(content, i); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(i.Dir, "influxdb.conf"), content.Bytes(), os.ModePerm); err != nil {
		return nil, errors.AddStack(err)
	}

	return i, nil
}

type prometheus struct {
	Host string
	Port int
	cmd  *exec.Cmd

	dir          string
	version      string
	InfluxAddr   string
	InfluxDBname string

	waitErr  error
	waitOnce sync.Once
}

func (m *prometheus) addr() string {
	return fmt.Sprintf("%s:%d", m.Host, m.Port)
}

func (m *prometheus) start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", m.Host, m.Port)
	args := []string{
		fmt.Sprintf("--config.file=%s", filepath.Join(m.dir, "prometheus.yml")),
		fmt.Sprintf("--web.external-url=http://%s", addr),
		fmt.Sprintf("--web.listen-address=%s", addr),
		fmt.Sprintf("--storage.tsdb.path=%s", filepath.Join(m.dir, "data")),
	}

	env := environment.GlobalEnv()
	params := &tiupexec.PrepareCommandParams{
		Ctx:         ctx,
		Component:   "prometheus",
		Version:     tiuputil.Version(m.version),
		InstanceDir: m.dir,
		WD:          m.dir,
		Args:        args,
		SysProcAttr: instance.SysProcAttr,
		Env:         env,
	}
	cmd, err := tiupexec.PrepareCommand(params)
	if err != nil {
		return err
	}
	cmd.Stdout = nil
	cmd.Stderr = nil

	m.cmd = cmd

	return m.cmd.Start()
}

func (m *prometheus) ready() bool {
	return m.cmd != nil
}

func (m *prometheus) wait() error {
	m.waitOnce.Do(func() {
		m.waitErr = m.cmd.Wait()
	})

	return m.waitErr
}

func (m *prometheus) getCmd() *exec.Cmd { return m.cmd }

func (m *prometheus) pid() int {
	if m.cmd != nil && m.cmd.Process != nil {
		return m.cmd.Process.Pid
	}
	return 0
}

// the cmd is not started after return
func newPrometheus(host, version, dir, influx string) (*prometheus, error) {
	dir = filepath.Join(dir, "prometheus")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.AddStack(err)
	}

	port, err := tiuputil.GetFreePort(host, 9090)
	if err != nil {
		return nil, err
	}

	m := new(prometheus)
	m.Host = host
	m.Port = port
	m.InfluxAddr = influx
	m.InfluxDBname = "diagcollector"
	m.dir = dir
	m.version = version

	const promCfg = `
global:
  scrape_interval:     15s # Set the scrape interval to every 15 seconds. Default is every 1 minute.
  evaluation_interval: 15s # Evaluate rules every 15 seconds. The default is every 1 minute.
  # scrape_timeout is set to the global default (10s).

# Load rules once and periodically evaluate them according to the global 'evaluation_interval'.
rule_files:
  # - "first_rules.yml"
  # - "second_rules.yml"

# A scrape configuration containing exactly one endpoint to scrape:
# Here it's Prometheus itself.
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
    - targets: ['{{.Host}}:{{.Port}}']

remote_read:
  - url: "http://{{.InfluxAddr}}/api/v1/prom/read?db={{.InfluxDBname}}"
    read_recent: true
`

	tmpl, err := template.New("prometheus.yml").Parse(promCfg)
	if err != nil {
		return nil, err
	}
	content := bytes.NewBufferString("")
	if err := tmpl.Execute(content, m); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(m.dir, "prometheus.yml"), content.Bytes(), os.ModePerm); err != nil {
		return nil, errors.AddStack(err)
	}

	return m, nil
}

type grafana struct {
	Host    string
	Port    int
	DataDir string
	Version string
	Cluster string
	Prom    string
	cmd     *exec.Cmd

	customFName string

	waitErr  error
	waitOnce sync.Once
}

func newGrafana(host, version, dir, clusterName, prom string) (*grafana, error) {
	if host == "" || host == "127.0.0.1" {
		host = "0.0.0.0"
	}

	g := &grafana{
		Host:    host,
		Version: version,
		DataDir: filepath.Join(dir, "grafana"),
		Cluster: clusterName,
		Prom:    prom,
	}

	var err error
	g.Port, err = tiuputil.GetFreePort(g.Host, 3000)
	if err != nil {
		return nil, err
	}

	fname := filepath.Join(g.DataDir, "conf", "provisioning", "dashboards", "dashboard.yml")
	err = writeDashboardConfig(fname, g.Cluster, filepath.Join(g.DataDir, "dashboards"))
	if err != nil {
		return nil, err
	}

	fname = filepath.Join(g.DataDir, "conf", "provisioning", "datasources", "datasource.yml")
	err = writeDatasourceConfig(fname, g.Cluster, g.Prom)
	if err != nil {
		return nil, err
	}

	tpl := `
[server]
# The ip address to bind to, empty will bind to all interfaces
http_addr = %s

# The http port to use
http_port = %d
`
	err = os.MkdirAll(filepath.Join(g.DataDir, "conf"), 0755)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	custom := fmt.Sprintf(tpl, g.Host, g.Port)
	customFName := filepath.Join(g.DataDir, "conf", "custom.ini")

	err = os.WriteFile(customFName, []byte(custom), 0644)
	if err != nil {
		return nil, errors.AddStack(err)
	}
	g.customFName = customFName

	return g, nil
}

// ref: https://grafana.com/docs/grafana/latest/administration/provisioning/
func writeDatasourceConfig(fname string, clusterName string, promURL string) error {
	err := makeSureDir(fname)
	if err != nil {
		return err
	}

	tpl := `apiVersion: 1
deleteDatasources:
  - name: %s
datasources:
  - name: %s
    type: prometheus
    access: proxy
    url: http://%s
    withCredentials: false
    isDefault: false
    tlsAuth: false
    tlsAuthWithCACert: false
    version: 1
    editable: true
`

	s := fmt.Sprintf(tpl, clusterName, clusterName, promURL)
	err = os.WriteFile(fname, []byte(s), 0644)
	if err != nil {
		return errors.AddStack(err)
	}

	return nil
}

// ref: templates/scripts/run_grafana.sh.tpl
// replace the data source in json to the one we are using.
func replaceDatasource(dashboardDir string, datasourceName string) error {
	// for "s/\${DS_.*-CLUSTER}/datasourceName/g
	re := regexp.MustCompile(`\${DS_.*-CLUSTER}`)

	err := filepath.Walk(dashboardDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("skip scan %s failed: %v", path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return errors.AddStack(err)
		}

		s := string(data)
		s = strings.ReplaceAll(s, "test-cluster", datasourceName)
		s = strings.ReplaceAll(s, "Test-Cluster", datasourceName)
		s = strings.ReplaceAll(s, "${DS_LIGHTNING}", datasourceName)
		s = re.ReplaceAllLiteralString(s, datasourceName)

		return os.WriteFile(path, []byte(s), 0644)
	})

	if err != nil {
		return err
	}

	return nil
}

func writeDashboardConfig(fname string, clusterName string, dir string) error {
	err := makeSureDir(fname)
	if err != nil {
		return err
	}

	tpl := `apiVersion: 1
providers:
  - name: %s
    folder: %s
    type: file
    disableDeletion: false
    editable: true
    updateIntervalSeconds: 30
    options:
      path: %s
`
	s := fmt.Sprintf(tpl, clusterName, clusterName, dir)

	err = os.WriteFile(fname, []byte(s), 0644)
	if err != nil {
		return errors.AddStack(err)
	}

	return nil
}

func makeSureDir(fname string) error {
	return os.MkdirAll(filepath.Dir(fname), 0755)
}

// dir should contains files untar the grafana.
// return not error iff the Cmd is started successfully.
func (g *grafana) start(ctx context.Context) (err error) {
	args := []string{
		"--homepath", g.DataDir,
		"--config", g.customFName,
		fmt.Sprintf("cfg:default.paths.logs=%s", path.Join(g.DataDir, "log")),
		fmt.Sprintf("cfg:default.paths.data=%s", path.Join(g.DataDir, "data")),
		fmt.Sprintf("cfg:default.paths.plugins=%s", path.Join(g.DataDir, "plugins")),
	}

	env := environment.GlobalEnv()
	params := &tiupexec.PrepareCommandParams{
		Ctx:         ctx,
		Component:   "grafana",
		Version:     tiuputil.Version(g.Version),
		InstanceDir: g.DataDir,
		WD:          g.DataDir,
		Args:        args,
		SysProcAttr: instance.SysProcAttr,
		Env:         env,
	}
	cmd, err := tiupexec.PrepareCommand(params)
	if err != nil {
		return err
	}
	cmd.Stdout = nil
	cmd.Stderr = nil

	g.cmd = cmd
	return g.cmd.Start()
}

func (g *grafana) ready() bool {
	return g.cmd != nil
}

func (g *grafana) wait() error {
	g.waitOnce.Do(func() {
		g.waitErr = g.cmd.Wait()
	})

	return g.waitErr
}

func (g *grafana) getCmd() *exec.Cmd { return g.cmd }

func (g *grafana) pid() int {
	if g.cmd != nil && g.cmd.Process != nil {
		return g.cmd.Process.Pid
	}
	return 0
}

func (g *grafana) addr() string { return fmt.Sprintf("%s:%d", g.Host, g.Port) }

func installIfMissing(component, version string) error {
	env := environment.GlobalEnv()

	installed, err := env.V1Repository().Local().ComponentInstalled(component, version)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	spec := repository.ComponentSpec{
		ID:      component,
		Version: version,
	}
	return env.V1Repository().UpdateComponents([]repository.ComponentSpec{spec})
}
