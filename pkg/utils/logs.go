package utils

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

// CollectLog collects logs
func CollectLog(collector, home, user, instanceID, inspectionID string, begin, end time.Time) error {
	cmd := exec.Command(
		collector,
		fmt.Sprintf("--home=%s", home),
		fmt.Sprintf("--instance-id=%s", instanceID),
		fmt.Sprintf("--inspection-id=%s", inspectionID),
		"--items=log",
		fmt.Sprintf("--begin=%s", begin.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+user,
		"CLUSTER_CREATE_TIME="+time.Now().Format(time.RFC3339), // it's not important
		"INSPECTION_TYPE=log",
	)
	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("run ", collector, ": ", err)
		return err
	}
	return nil
}
