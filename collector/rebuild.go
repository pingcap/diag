// Copyright 2020 PingCAP, Inc.
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

	"github.com/pingcap/errors"
	"github.com/pingcap/tiup/components/playground/instance"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	"github.com/pingcap/tiup/pkg/environment"
	tiupexec "github.com/pingcap/tiup/pkg/exec"
	"github.com/pingcap/tiup/pkg/localdata"
	"github.com/pingcap/tiup/pkg/repository"
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

	// read clsuter name
	clsNameFile, err := os.ReadFile(path.Join(dataDir, fileNameClusterName))
	if err != nil {
		return err
	}
	clsName := string(clsNameFile)

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
		}
	}()

	bootErr := p.boot(ctx, dataDir, opt.Host, clsVer, clsName)
	if bootErr != nil {
		// always kill all process started and wait before quit.
		atomic.StoreInt32(&p.lastSig, int32(syscall.SIGKILL))
		p.terminate(syscall.SIGKILL)
		_ = p.wait()
		return errors.Annotate(bootErr, "Bootstrapping failed")
	}

	atomic.StoreUint32(&booted, 1)

	waitErr := p.wait()
	if waitErr != nil {
		return waitErr
	}
	return nil
}

type component interface {
	start() error
	wait() error
	getCmd() *exec.Cmd
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

func (b *rebuilder) boot(ctx context.Context, dataDir, host, clsVer, clsName string) error {
	// prepare influxdb
	if err := installIfMissing("influxdb", clsVer); err != nil {
		return err
	}
	var influxAddr string
	if insInflux, err := newInfluxdb(ctx, host, clsVer, dataDir); err == nil {
		b.Proc["influxdb"] = insInflux
		influxAddr = insInflux.addr()
	} else {
		return err
	}

	// prepare prometheus
	var promAddr string
	if err := installIfMissing("prometheus", clsVer); err != nil {
		return err
	}
	if insProm, err := newPrometheus(ctx, host, clsVer, dataDir, influxAddr); err == nil {
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
	b.Proc["grafana"] = newGrafana(ctx, host, clsVer, dataDir, clsName, promAddr)

	for comp, ins := range b.Proc {
		fmt.Printf("setting up %s...", comp)
		i := ins
		c := comp
		b.walker.Go(func() error {
			if err := i.start(); err != nil {
				defer b.terminate(syscall.SIGKILL)
				return err
			}
			fmt.Printf("%s started.", c)
			return nil
		})
	}
	fmt.Println("finished setting up monitoring system on localhost.")

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
		pid := inst.getCmd().ProcessState.Pid()
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
	host     string
	bindPort int
	httpPort int
	dir      string
	cmd      *exec.Cmd

	waitErr  error
	waitOnce sync.Once
}

func (i *influxdb) start() error {
	return i.cmd.Start()
}

func (i *influxdb) wait() error {
	i.waitOnce.Do(func() {
		i.waitErr = i.cmd.Wait()
	})

	return i.waitErr
}

func (i *influxdb) getCmd() *exec.Cmd { return i.cmd }

func (i *influxdb) addr() string {
	return fmt.Sprintf("%s:%d", i.host, i.httpPort)
}

// the cmd is not started after return
func newInfluxdb(ctx context.Context, host, version, dir string) (*influxdb, error) {
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
	i.host = host
	i.dir = dir
	i.bindPort = bindPort
	i.httpPort = httpPort

	args := []string{
		fmt.Sprintf("-config %s", filepath.Join(dir, "influxdb.conf")),
	}

	env := environment.GlobalEnv()
	params := &tiupexec.PrepareCommandParams{
		Ctx:         ctx,
		Component:   "influxdb",
		Version:     tiuputil.Version(version),
		InstanceDir: dir,
		WD:          dir,
		Args:        args,
		SysProcAttr: instance.SysProcAttr,
		Env:         env,
	}
	cmd, err := tiupexec.PrepareCommand(params)
	if err != nil {
		return nil, err
	}

	i.cmd = cmd

	const influxCfg = `
bind-address = "{{.host}}:{{.bindPort}}"
[meta]
	dir = "{{.dir}}/meta"
[data]
	dir = "{{.dir}}/data"
	wal-dir = "{{.dir}}/wal"
	series-id-set-cache-size = 100
[coordinator]
[retention]
[shard-precreation]
[monitor]
[http]
	bind-address = "{{.host}}:{{.httpPort}}"
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

	if err := os.WriteFile(filepath.Join(dir, "influxdb.conf"), content.Bytes(), os.ModePerm); err != nil {
		return nil, errors.AddStack(err)
	}

	return i, nil
}

type prometheus struct {
	host string
	port int
	cmd  *exec.Cmd

	influxAddr   string
	influxDBname string

	waitErr  error
	waitOnce sync.Once
}

func (m *prometheus) addr() string {
	return fmt.Sprintf("%s:%d", m.host, m.port)
}

func (m *prometheus) start() error {
	return m.cmd.Start()
}

func (m *prometheus) wait() error {
	m.waitOnce.Do(func() {
		m.waitErr = m.cmd.Wait()
	})

	return m.waitErr
}

func (m *prometheus) getCmd() *exec.Cmd { return m.cmd }

// the cmd is not started after return
func newPrometheus(ctx context.Context, host, version, dir, influx string) (*prometheus, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.AddStack(err)
	}

	port, err := tiuputil.GetFreePort(host, 9090)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	m := new(prometheus)
	m.host = host
	m.port = port
	m.influxAddr = influx
	m.influxDBname = "diagcollector"

	args := []string{
		fmt.Sprintf("--config.file=%s", filepath.Join(dir, "prometheus.yml")),
		fmt.Sprintf("--web.external-url=http://%s", addr),
		fmt.Sprintf("--web.listen-address=%s:%d", host, port),
		fmt.Sprintf("--storage.tsdb.path=%s", filepath.Join(dir, "data")),
	}

	env := environment.GlobalEnv()
	params := &tiupexec.PrepareCommandParams{
		Ctx:         ctx,
		Component:   "prometheus",
		Version:     tiuputil.Version(version),
		InstanceDir: dir,
		WD:          dir,
		Args:        args,
		SysProcAttr: instance.SysProcAttr,
		Env:         env,
	}
	cmd, err := tiupexec.PrepareCommand(params)
	if err != nil {
		return nil, err
	}

	m.cmd = cmd

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
    - targets: ['{{.host}}:{{.port}}']

remote_read:
  - url: "http://{{.influxAddr}}/api/v1/prom/read?db={{.influxDBname}}"
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

	if err := os.WriteFile(filepath.Join(dir, "prometheus.yml"), content.Bytes(), os.ModePerm); err != nil {
		return nil, errors.AddStack(err)
	}

	return m, nil
}

type grafana struct {
	host    string
	port    int
	cmd     *exec.Cmd
	dataDir string
	version string
	cluster string
	prom    string

	ctx      context.Context
	waitErr  error
	waitOnce sync.Once
}

func newGrafana(ctx context.Context, host, version, dir, clusterName, prom string) *grafana {
	return &grafana{
		host:    host,
		version: version,
		dataDir: dir,
		cluster: clusterName,
		prom:    prom,
		ctx:     ctx,
	}
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
    url: %s
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
func (g *grafana) start() (err error) {
	g.port, err = tiuputil.GetFreePort(g.host, 3000)
	if err != nil {
		return err
	}

	fname := filepath.Join(g.dataDir, "conf", "provisioning", "dashboards", "dashboard.yml")
	err = writeDashboardConfig(fname, g.cluster, filepath.Join(g.dataDir, "dashboards"))
	if err != nil {
		return err
	}

	fname = filepath.Join(g.dataDir, "conf", "provisioning", "datasources", "datasource.yml")
	err = writeDatasourceConfig(fname, g.cluster, g.prom)
	if err != nil {
		return err
	}

	tpl := `
[server]
# The ip address to bind to, empty will bind to all interfaces
http_addr = %s

# The http port to use
http_port = %d
`
	err = os.MkdirAll(filepath.Join(g.dataDir, "conf"), 0755)
	if err != nil {
		return errors.AddStack(err)
	}

	custome := fmt.Sprintf(tpl, g.host, g.port)
	customeFName := filepath.Join(g.dataDir, "conf", "custom.ini")

	err = os.WriteFile(customeFName, []byte(custome), 0644)
	if err != nil {
		return errors.AddStack(err)
	}

	args := []string{
		"--homepath",
		g.dataDir,
		"--config",
		customeFName,
		fmt.Sprintf("cfg:default.paths.logs=%s", path.Join(g.dataDir, "log")),
	}

	env := environment.GlobalEnv()
	params := &tiupexec.PrepareCommandParams{
		Ctx:         g.ctx,
		Component:   "grafana",
		Version:     tiuputil.Version(g.version),
		InstanceDir: g.dataDir,
		WD:          g.dataDir,
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

func (g *grafana) wait() error {
	g.waitOnce.Do(func() {
		g.waitErr = g.cmd.Wait()
	})

	return g.waitErr
}

func (g *grafana) getCmd() *exec.Cmd { return g.cmd }

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
