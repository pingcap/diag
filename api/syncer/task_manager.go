package syncer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type TaskManager struct {
	Tasks      sync.Map
	TodoTaskCh chan SyncTask
	Cfg        RsyncConfig
	Interval   time.Duration
	StopCh     chan struct{}
}

func (t *TaskManager) CancelAllTask() {
	t.Tasks.Range(func(k, v interface{}) bool {
		if task, ok := v.(SyncTask); ok {
			task.Cancel()
			t.Tasks.Delete(task.Key)
		}
		return true
	})
}

func (t *TaskManager) RunTasks(tasks []SyncTask) {
	// Todo: diff list of new and old tasks, add or delete tasks according to the situation
	// Newly added task: Added to task list and passed into TodoTaskCh
	// Deleted Task: Cancel the old task and delete this task from the task list
	// Modified Task: Cancel the old task, delete this task from the task list,
	//                and pass the new task into TodoTaskCh

	// Current practice:
	// cancel all old tasks and add all new ones
	t.CancelAllTask()
	for _, task := range tasks {
		log.Infof("start a new sync task. key=%s", task.Key)
		go func(task SyncTask) {
			t.TodoTaskCh <- task
		}(task)
	}
}

func (t *TaskManager) Start() {
	// MainLoop
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case task := <-t.TodoTaskCh:
			go t.execCommand(task)
		case <-ticker.C:
			// Prevent deadlock when syncTasks is empty
			continue
		case <-t.StopCh:
			// only for test
			return
		}
	}
}

func (t *TaskManager) Stop() {
	fmt.Println("stop tasks manager")
	t.StopCh <- struct{}{}
}

func buildCmdWithCancel(task SyncTask, cfg RsyncConfig) (*exec.Cmd, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	for _, pattern := range task.Filters {
		cfg.Args = append(cfg.Args, fmt.Sprintf(`--include=%s`, pattern))
	}
	cfg.Args = append(cfg.Args, `--exclude=*`)
	args := append(cfg.Args, task.From, task.To)
	cmd := exec.CommandContext(ctx, "rsync", args...)
	return cmd, ctx, cancel
}

func (t *TaskManager) execCommand(task SyncTask) {
	cmd, ctx, cancel := buildCmdWithCancel(task, t.Cfg)
	task.CancelFunc = cancel
	task.Ctx = ctx
	t.Tasks.Store(task.Key, task)
	err := os.MkdirAll(task.To, os.ModePerm)
	if err != nil {
		log.Errorf("failed to create folder %s ", task.To)
		return
	}
	err = cmd.Start()
	if err != nil {
		log.Errorf("failed to start command %s ", task.Key)
		return
	}
	// done indicate the reason for task quitting
	// true: this task finished normally
	// false: this task has been canceled
	done := make(chan bool)
	// Wait for rsync execution finished ,
	// then wait for an while (specified by interval),
	// notify done channel.
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("task stopped: %s err=%s", task.Key, err)
			done <- true
			return
		}
		time.Sleep(t.Interval)
		done <- false
	}()
	select {
	// If the task is cancelled,
	// kill rsync process,
	// and quit current task
	case <-task.Ctx.Done():
		err := task.Ctx.Err()
		if err == context.Canceled {
			log.Printf("cancel task key=%s", task.Key)
		} else {
			log.Error(err)
		}
		if cmd.ProcessState.Exited() {
			return
		}
		err = cmd.Process.Kill()
		if err != nil {
			log.Errorf("failed to kill rsync progress. key=%s err=%s", task.Key, err)
		}
		return
	// If the rsync process has completed, restart a new synchronization task
	case finished, ok := <-done:
		if ok && !finished {
			t.TodoTaskCh <- task
		}
	}
}
