package task

import (
	"os/exec"
	"path"

	log "github.com/sirupsen/logrus"
)

type SaveLogTask struct {
	BaseTask
}

func SaveLog(base BaseTask) Task {
	return &SaveLogTask{base}
}

func (t *SaveLogTask) Run() error {
	if !t.data.args.Collect(ITEM_LOG) || t.data.status[ITEM_LOG].Status != "success" {
		return nil
	}

	if err := exec.Command(
		"cp", "-r", path.Join(t.src, "log"),
		path.Join(t.home, "remote-log", t.inspectionId),
	).Run(); err != nil {
		log.Error("copy logs:", err)
		t.InsertSymptom("exception", "failed to copy logs", "check your disk and filesystem please.")
		return err
	}

	return nil
}
