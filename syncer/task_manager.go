package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

type TaskManager struct {
	Cfg      RsyncConfig
	Interval time.Duration
	StopCh   chan struct{}
	cancel   context.CancelFunc
}

func (t *TaskManager) CancelAllTask() {
	if t.cancel != nil {
		log.Info("try to cancel all task")
		t.cancel()
	}
}

func (t *TaskManager) RunTasks(tasks []SyncTask) {
	// cancel all old tasks and add all new ones
	log.Info("RunTasks")
	t.CancelAllTask()
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	for _, task := range tasks {
		//log.Infof("start a new sync task. key=%s", task.Key)
		go func(task SyncTask) {
			t.TaskLoop(ctx, task)
		}(task)
	}
}

func (t *TaskManager) TaskLoop(ctx context.Context, task SyncTask) {
	cmd := buildCmd(ctx, task, t.Cfg)
	for t.execCommand(ctx, cmd, task) {
		cmd = buildCmd(ctx, task, t.Cfg)
	}
}

func (t *TaskManager) Stop() {
	log.Println("stop tasks manager")
	t.StopCh <- struct{}{}
}

func buildCmd(ctx context.Context, task SyncTask, cfg RsyncConfig) *exec.Cmd {
	for _, pattern := range task.Filters {
		cfg.Args = append(cfg.Args, fmt.Sprintf(`--include=%s`, pattern))
	}
	cfg.Args = append(cfg.Args, `--exclude=*`)
	args := append(cfg.Args, task.From, task.To)
	cmd := exec.CommandContext(ctx, "rsync", args...)
	return cmd
}

func (t *TaskManager) execCommand(ctx context.Context, cmd *exec.Cmd, task SyncTask) (continueFlag bool) {
	continueFlag = false
	err := os.MkdirAll(task.To, os.ModePerm)
	if err != nil {
		log.Errorf("failed to create folder %s ", task.To)
		return
	}
	log.Infoln("task start: key=", task.Key)
	err = cmd.Start()
	if err != nil {
		log.Errorf("failed to start command %s ", task.Key)
		return
	}
	// Wait for rsync execution finished ,
	// then wait for an while (specified by interval),
	// notify done channel.
	err = cmd.Wait()
	if err != nil {
		// if canceled, return immediately
		log.Warnf("task stopped: %s err=%s", task.Key, err)
		return
	}
	timer := time.NewTimer(t.Interval)
	select {
	case <-ctx.Done():
		log.Warnf("task canceled: %s err=%s", task.Key, ctx.Err())
		return
	case <-timer.C:
		return true
	}
}
