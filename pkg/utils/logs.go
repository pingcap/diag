// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
	args := []string{
		fmt.Sprintf("--home=%s", home),
		fmt.Sprintf("--instance-id=%s", instanceID),
		fmt.Sprintf("--inspection-id=%s", inspectionID),
		"--items=log",
		fmt.Sprintf("--begin=%s", begin.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
	}
	cmd := exec.Command(collector, args...)
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
