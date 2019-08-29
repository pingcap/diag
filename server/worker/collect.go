package worker

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

func (w *worker) Collect(inspectionId, inspectionType string, config *model.Config) error {
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

	items := []string{"metric", "basic", "dbinfo", "config", "log"}
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
		if config.CollectDemsg {
			//	items = append(items, "demsg")
		}
	}

	cmd := exec.Command(
		w.c.Collector,
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(w.c.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(w.c.Home, "inspection")),
		fmt.Sprintf("--collect=%s", strings.Join(items, ",")),
		fmt.Sprintf("--log-dir=%s", path.Join(w.c.Home, "remote-log", instanceId)),
		fmt.Sprintf("--log-spliter=%s", path.Join(w.c.Home, "bin", "spliter")),
		fmt.Sprintf("--begin=%s", from.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", to.Format(time.RFC3339)),
	)
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+w.c.User.Name,
		"CLUSTER_CREATE_TIME="+instance.CreateTime.Format(time.RFC3339),
		"INSPECTION_TYPE="+inspectionType,
	)
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
