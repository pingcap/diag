package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

/*
* Normally this function does not return
* It should watch the directory specified by topoDir, which holds a json file of topology.json for each cluster.
* These json files indicate which hosts the cluster contains and the deployment directory of the component in these hosts
* This function needs to synchronize {user}@ip:{deploy}/log to
* the directory specified by targetDir through rsync for each machine.
* We need to be able to distinguish which logs belong to which clusters and which components,
* the synchronization interval is specified by interval,
* and the bandwidth limit for synchronization is specified by bwlimit
* (the rsync command line parameter supports bwlimit.)
 */

func Sync(topoDir string, targetDir string, interval time.Duration, bwlimit int) error {
	watcher := Watcher{
		topoDir,
		targetDir,
	}
	manager := TaskManager{
		Interval:   interval,
		StopCh:     make(chan struct{}),
		Cfg: RsyncConfig{
			Args: []string{
				"-avz",
				fmt.Sprintf("--bwlimit=%d", bwlimit),
			},
		},
	}
	tasks, err := watcher.LoadTasks()
	if err != nil {
		log.Errorf("failed to load tasks: %s", err)
	}

	// Receive the new task list,
	// cancel all the old task,
	// and run the newly added task
	manager.RunTasks(tasks)
	// Watch specifies the directory,
	// rescans all files when changes occur,
	// build a new rsync task list,
	// and passes it to the TaskManager.runTasks()
	watcher.Watch(manager)
	return nil
}

type RsyncConfig struct {
	Args []string
	From string
	To   string
}

type Cluster struct {
	Name    string `json:"cluster_name"`
	Status  string
	Message string
	Hosts   []Host
}

type Host struct {
	Status     string
	Ip         string
	EnableSudo bool `json:"enable_sudo"`
	User       string
	Components []Component
	Message    string
}

type Component struct {
	Status    string
	DeployDir string `json:"deploy_dir"`
	Name      string
	Port      string
}

// NewCluster return a new Cluster
func NewCluster(fileName string) (*Cluster, error) {
	c := &Cluster{}
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	b, _ := ioutil.ReadAll(f)
	err = json.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// LoadTask construct the SyncTask slice according to the Cluster
func (c *Cluster) LoadTasks(targetDir, uuid string) []SyncTask {
	var tasks []SyncTask
	for _, host := range c.Hosts {
		if host.Status != "success" {
			continue
		}
		for _, component := range host.Components {
			// {cluster_uuid}_{host_ip}_{component_name}_{component_port}
			key := fmt.Sprintf("%s_%s_%s_%s", uuid, host.Ip, component.Name, component.Port)
			// {user}@{host_ip}:{deploy_dir}/log
			from := fmt.Sprintf("%s@%s:%s/log/", host.User, host.Ip, component.DeployDir)
			// {target_dir}/{cluster_uuid}/{host_ip}/{component_name}-{component_port}
			folderName := fmt.Sprintf("%s-%s", component.Name, component.Port)
			to := path.Join(targetDir, uuid, host.Ip, folderName)
			// log file filter pattern, e.g. "tikv*"
			filters := []string{PatternStr(component.Name)}
			if patterns, ok := componentPattern[component.Name]; ok {
				for _, filename := range patterns {
					filters = append(filters, PatternStr(filename))
				}
			}
			task := SyncTask{
				Key:     key,
				From:    from,
				To:      to,
				Filters: filters,
			}
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// PatternStr return the pattern string used by rsync parameter: --filter
func PatternStr(filename string) string {
	return filename + "*"
}

// syncTask inform a pair of folder path
// rsync uses them as parameters
// Key:        component unique identification
// From:       src_path
// To:         dist_path
// Filters:    log file name pattern
// Ctx:        used for receive the cancel signal
// CancelFunc: used for cancel the task
type SyncTask struct {
	Key     string
	From    string
	To      string
	Filters []string
	cmd     *exec.Cmd
}
