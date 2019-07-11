package syncer

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

/*
 * 正常情况下该函数不返回
 * 它应该watch topoDir指定的目录，该目录为每一个集群存放了一个topology.json的json文件
 * 这些json文件指示了该集群包含哪些机器，以及该集群的deploy目录
 * 该函数需要针对每台机器，通过rsync将{user}@ip:{deploy}/log同步到targetDir指定的目录中，同步之后
 * 需要能够区分哪些日志属于哪些集群以及哪些组件，同步的时间间隔为interval指定的间隔，同步的带宽限制由bwlimit指定
 * rsync命令行参数支持bwlimit
 */

type rsyncConfig struct {
	Args []string
	From string
	To   string
}

type cluster struct {
	Name    string `json:"cluster_name"`
	Status  string
	Message string
	Hosts   []host
}

type host struct {
	Status     string
	Ip         string
	EnableSudo bool `json:"enable_sudo"`
	User       string
	Components []component
	Message    string
}

type component struct {
	Status    string
	DeployDir string `json:"deploy_dir"`
	Name      string
	Port string       // []string
}

func Sync(topoDir string, targetDir string, interval time.Duration, bwlimit int) error {
	// watch 指定目录，当改变时，解析文件，然后调用 rsync
	rsyncCfg := rsyncConfig{
		Args: []string{"-avz", fmt.Sprintf("--bwlimit=%d", bwlimit)},
	}
	err := watchDir(topoDir, targetDir, rsyncCfg, interval)
	if err != nil {
		return err
	}

	return nil
}

func watchDir(topoDir string, targetDir string, rsyncCfg rsyncConfig, interval time.Duration) error {
	var g errgroup.Group
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	g.Go(func() error {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return errors.New("failed to receive watcher events")
				}
				log.Println("event:", event)
				if (event.Op&fsnotify.Write == fsnotify.Write) ||
					(event.Op&fsnotify.Create == fsnotify.Create) {
					log.Println("topology file modified:", event.Name)
					changedFile := event.Name
					uuid := strings.TrimSuffix(changedFile, filepath.Ext(changedFile))
					c := cluster{}

					// 解析 json 文件
					err := c.parseFile(changedFile)
					if err != nil {
						return err
					}

					// 需要同步的机器（和路径）和对应的 deploy 目录
					syncTasks := c.parseSyncTasks(targetDir, uuid)

					// 调用 rsync 进行同步
					err = callRsync(syncTasks, rsyncCfg)
					if err != nil {
						return err
					}
					// 休息一段时间，再重新重新回到这个流程
					time.Sleep(interval)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return err
				}
			}
		}
	})

	err = watcher.Add(topoDir)
	if err != nil {
		return err
	}
	return g.Wait()
}

func (c *cluster) parseFile(fileName string) error {
	f, err := os.Open(fileName)
	if err != nil {
		return err
	}
	b, _ := ioutil.ReadAll(f)
	return json.Unmarshal(b, &c)
}

// syncTasks inform a pair of folder path
// rsync use them to collect log from each host to localhost
// Key:   {cluster_uuid}_{host_ip}_{component_name}_{component_port}
// Value: syncTask {
//		From:    src_path
//		To:      dist_path
//		Filters: log file name pattern
// }
type syncTasks map[string]syncTask

type syncTask struct {
	From string
	To string
	Filters []string
	Status string
}

func (c *cluster) parseSyncTasks(targetDir string, uuid string) syncTasks {
	tasks := make(syncTasks)
	// 构造去重的 syncTasks
	for _, host := range c.Hosts {
		if host.Status != "success" {
			continue
		}
		for _, component := range host.Components {
			// {cluster_uuid}_{host_ip}_{component_name}_{component_port}
			key := fmt.Sprintf("%s_%s_%s_%s", uuid, host.Ip, component.Name,component.Port)
			// {user}@{host_ip}:{deploy_dir}/log
			from := fmt.Sprintf("%s@%s:%s/log/", host.User, host.Ip, component.DeployDir)

			folderName := fmt.Sprintf("%s-%s", component.Name, component.Port)
			// {target_dir}/{cluster_uuid}/{host_ip}/{component_name}-{component_port}
			to := path.Join(targetDir, uuid, host.Ip, folderName)
			// log file filter pattern, e.g. "tikv*"
			filters := []string{component.Name + "*"}
			if pattern, ok := componentPattern[component.Name]; ok {
				filters = append(filters, pattern + "*")
			}
			tasks[key] = syncTask{
				From: from,
				To: to,
				Filters: filters,
			}
		}
	}
	return tasks
}

func callRsync(tasks syncTasks, cfg rsyncConfig) error {
	var g errgroup.Group
	for _, task := range tasks {
		g.Go(func() error {
			for _, v := range task.Filters {
				cfg.Args = append(cfg.Args, fmt.Sprintf("--include=\"%s\"", v))
			}
			args := append(cfg.Args, task.From, task.To)
			cmd := exec.Command("rsync", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				return err
			}
			return nil
		})
	}

	return g.Wait()
}
