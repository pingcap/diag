package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
)

func CollectLog(collector, home, user, instanceId, inspectionId string, begin, end time.Time) error {
	cmd := exec.Command(
		collector,
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(home, "inspection")),
		"--collect=log",
		fmt.Sprintf("--log-dir=%s", path.Join(home, "remote-log", instanceId)),
		fmt.Sprintf("--log-spliter=%s", path.Join(home, "bin", "spliter")),
		fmt.Sprintf("--begin=%s", begin.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+user,
		"INSPECTION_TYPE=log",
	)
	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("run ", collector, ": ", err)
		return err
	}
	return nil
}
