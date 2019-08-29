package scheduler

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/server/worker"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

type Scheduler interface {
	Reload() error
}

type scheduler struct {
	m model.Model
	c *cron.Cron
	w worker.Worker
}

func New(m model.Model, w worker.Worker) Scheduler {
	return &scheduler{
		m: m,
		w: w,
	}
}

func (s *scheduler) Reload() error {
	if s.c != nil {
		s.c.Stop()
	}

	s.c = cron.New()

	configs, err := s.m.ListInstanceConfig()
	if err != nil {
		return fmt.Errorf("list configs for all instance: %s", err)
	}

	for _, config := range configs {
		spec, err := quartz(config)
		if err != nil {
			log.Errorf("parse quantz from config of instance %s: %s", config.InstanceId, err)
			continue
		}
		config.SchedRange = []time.Time{
			time.Now().Add(time.Duration(-config.AutoSchedDuration) * time.Hour),
			time.Now(),
		}
		if err := s.c.AddFunc(spec, func() {
			inspectionId := uuid.New().String()
			if err := s.w.Collect(inspectionId, "auto", config); err != nil {
				log.Errorf("diagnose %s: %s", inspectionId, err)
			} else if err := s.w.Analyze(inspectionId); err != nil {
				log.Errorf("analyze %s: %s", inspectionId, err)
			}
		}); err != nil {
			log.Errorf("add auto sched task for instance %s: %s", config.InstanceId, err)
		} else {
			log.Info("add auto sched task success, quartz expression:", spec)
		}
	}

	s.c.Start()
	return nil
}

func quartz(c *model.Config) (string, error) {
	ss := strings.Split(c.AutoSchedStart, ":")
	if len(ss) != 2 {
		return "", fmt.Errorf("config(auto_sched_start) is valid for instance: %s", c.InstanceId)
	}
	return fmt.Sprintf("0 %s %s * * %s", ss[1], ss[0], c.AutoSchedDay), nil
}
