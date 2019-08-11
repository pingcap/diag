package logs

import (
	"os"
	"os/exec"
	"path"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type copyLogTask struct{}

func CopyLogs() *copyLogTask {
	return &copyLogTask{}
}

// Copy logs from inspection directory to remote-log directory.
// Because the log searcher only search remote-log directory.
func (t *copyLogTask) Run(c *boot.Config, db *boot.DB) {
	if _, err := os.Stat(path.Join(c.Src, "log")); err != nil {
		if !os.IsNotExist(err) {
			log.Error("read log dir:", err)
		}
		return
	}

	if err := exec.Command(
		"cp", "-r", path.Join(c.Src, "log"),
		path.Join(c.Home, "remote-log", c.InspectionId),
	).Run(); err != nil {
		log.Error("copy logs:", err)
		db.InsertSymptom(c.InspectionId, "exception", "failed to copy logs", "check your disk and filesystem please.")
	}
}
