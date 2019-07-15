package syncer

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"
)

/*
 * 正常情况下该函数不返回
 * 它应该watch topoDir指定的目录，该目录为每一个集群存放了一个topology.json的json文件
 * 这些json文件指示了该集群包含哪些机器，以及该集群的deploy目录
 * 该函数需要针对每台机器，通过rsync将{user}@ip:{deploy}/log同步到targetDir指定的目录中，同步之后
 * 需要能够区分哪些日志属于哪些集群以及哪些组件，同步的时间间隔为interval指定的间隔，同步的带宽限制由bwlimit指定
 * rsync命令行参数支持bwlimit
 */

type RsyncConfig struct {
	Args []string
	From string
	To   string
}

func Sync(topoDir string, targetDir string, interval time.Duration, bwlimit int) error {
	watcher := Watcher{
		topoDir,
		targetDir,
	}
	manager := TaskManager{
		Interval: interval,
		Cfg: RsyncConfig{
			Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", bwlimit)},
		},
	}
	// watch 指定目录，发生改变时，重新扫描所有文件，构造新的 rsync 任务列表，传递给 manager
	err := watcher.watch(manager)
	if err != nil {
		return err
	}
	// 解析新的任务列表，cancel 旧任务，运行新添加任务
	go manager.Start()
	return nil
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

func PatternStr(filename string) string {
	return filename + "*"
}

// syncTask inform a pair of folder path
// rsync use them to collect log from each Host to localhost
//      Key:     Component identity
//		From:    src_path
//		To:      dist_path
//		Filters: log file name pattern
// }

type SyncTask struct {
	Key        string
	From       string
	To         string
	Filters    []string
	Ctx        context.Context
	CancelFunc context.CancelFunc
}

func (s *SyncTask) Cancel() {
	s.CancelFunc()
}
