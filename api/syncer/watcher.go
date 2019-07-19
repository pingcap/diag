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

// Watch watch the topoDir, resolve all tasks, and pass them to taskManager
func (w *Watcher) Watch(taskManager TaskManager) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	go func() {
		for {
			select {
			// TODO: If there are more than one events, only the latest one will be taken out.
			case event, ok := <-watcher.Events:
				if !ok {
					log.Error("Watcher events channel closed")
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Write) == 0 {
					continue
				}
				log.Println("topology file modified:", event.Name)
				// If one of the files is modified, the entire directory will be rescanned.
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
	err = watcher.Add(w.targetDir)
	if err != nil {
		return err
	}
	return nil
}

// LoadTask return all SyncTasks in current topology folder
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
		tasks := cluster.LoadTasks(w.targetDir, uuid)
		allTasks = append(allTasks, tasks...)
	}
	return allTasks, nil
}
