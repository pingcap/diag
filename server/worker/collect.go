package worker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

func (w *worker) Collect(inspectionId string, config *model.Config, extraEnv map[string]string) error {
	instanceId := config.InstanceId
	instance, err := w.m.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return err
	}

	from := time.Now().Add(time.Duration(-10) * time.Minute)
	to := time.Now()
	if len(config.SchedRange) > 0 {
		from = config.SchedRange[0]
	}
	if len(config.SchedRange) > 1 {
		to = config.SchedRange[1]
	}
	// TODO: put dmesg as a selectable arguments.
	items := []string{"alert", "basic", "config", "dbinfo", "log", "metric", "network"}
	if config != nil {
		if config.CollectHardwareInfo {
			//	items = append(items, "hardware")
		}
		if config.CollectSoftwareInfo {
			//	items = append(items, "software")
		}
		if config.CollectLog {
			//	items = append(items, "log")
		}
		if config.CollectDmesg {
			items = append(items, "dmesg")
		}
	}

	cmd := exec.Command(
		w.c.Collector,
		fmt.Sprintf("--home=%s", w.c.Home),
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--items=%s", strings.Join(items, ",")),
		fmt.Sprintf("--begin=%s", from.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", to.Format(time.RFC3339)),
	)
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+w.c.User.Name,
		"CLUSTER_CREATE_TIME="+instance.CreateTime.Format(time.RFC3339),
	)
	for k, v := range extraEnv {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)
	err = cmd.Run()
	if err != nil {
		log.Error("run ", w.c.Collector, ": ", err)
		return err
	}
	return nil
}
