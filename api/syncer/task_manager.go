package syncer

import (
	"context"
	"fmt"
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
	// Todo: diff 新旧任务列表，分情况增删任务
	// 新添加的任务：添加到任务列表，并传入 TodoTaskCh 中
	// 被删除的任务：Cancel 旧任务，任务列表中删除这个任务
	// 被修改的任务：Cancel 旧任务，任务列表中删除这个任务，添加新任务到任务列表中，并传入 TodoTaskCh

	// 目前的做法：
	// cancel 所有旧任务，再添加所有新的任务
	t.CancelAllTask()
	for _, task := range tasks {
		t.TodoTaskCh <- task
	}
}

func (t *TaskManager) Start() {
	// MainLoop
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case task := <-t.TodoTaskCh:
			cmd, ctx, cancel := buildCmdWithCancel(task, t.Cfg)
			task.CancelFunc = cancel
			task.Ctx = ctx
			t.Tasks.Store(task.Key, task)
			go execCommand(cmd, task, t.Interval, t.TodoTaskCh)
		case <-ticker.C:
			continue
		}
	}
}

func buildCmdWithCancel(task SyncTask, cfg RsyncConfig) (*exec.Cmd, context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	for _, pattern := range task.Filters {
		cfg.Args = append(cfg.Args, fmt.Sprintf("--include=\"%s\"", pattern))
	}
	args := append(cfg.Args, task.From, task.To)
	cmd := exec.CommandContext(ctx, "rsync", args...)
	return cmd, ctx, cancel
}

func execCommand(cmd *exec.Cmd, task SyncTask, interval time.Duration, todoTaskCh chan SyncTask) {
	err := cmd.Start()
	if err != nil {
		log.Errorf("failed to start command %s ", task.Key)
		return
	}
	done := make(chan struct{})
	// 等待 rsync 执行结束，之后，再等待一段时间（Interval），通知 done channel
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Errorf("task stopped: %s err=%s", task.Key, err)
			return
		}
		time.Sleep(interval)
		done <- struct{}{}
	}()
	select {
	// 如果该任务被 cancel 了，kill rsync 进程，直接 return
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
	// 如果 rsync 进程已经结束，重新开始一次同步
	case <-done:
		todoTaskCh <- task
	}
}
