package syncer

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

type Watcher struct {
	topoDir   string
	targetDir string
}

func (w *Watcher) watch(taskManager TaskManager) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					log.Error("Watcher events channel closed")
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Write) == 0 {
					continue
				}
				log.Println("topology file modified:", event.Name)
				// 其中一个文件发生修改，整个目录重新扫描一遍
				tasks, err := w.LoadTasks()
				if err != nil {
					log.Errorf("failed to load tasks: %s", err)
					continue
				}
				taskManager.RunTasks(tasks)
			case err, ok := <-watcher.Errors:
				if !ok {
					log.Errorf("failed to watch topology dir: %s", err)
				}
			}
		}
	}()
	return nil
}

func (w *Watcher) LoadTasks() ([]SyncTask, error) {
	var allTasks []SyncTask
	dir, err := ioutil.ReadDir(w.topoDir)
	if err != nil {
		return nil, err
	}
	for _, fi := range dir {
		if fi.IsDir() {
			continue
		}
		fileName := fi.Name()
		ext := filepath.Ext(fileName)
		if ext != ".json" {
			continue
		}
		uuid := strings.TrimSuffix(fileName, ext)

		cluster, err := NewCluster(fileName)
		if err != nil {
			return nil, err
		}
		// 获取需要同步的机器（和路径）和对应的 deploy 目录
		tasks := cluster.LoadTasks(w.targetDir, uuid)
		allTasks = append(allTasks, tasks...)
	}
	return allTasks, nil
}
