package task

import (
	log "github.com/sirupsen/logrus"
)

type AnalyzeTask struct {
	BaseTask
}

func Analyze(base BaseTask) Task {
	return &AnalyzeTask{base}
}

// TODO
func (t *AnalyzeTask) Run() error {
	if _, err := t.db.Exec(
		`UPDATE inspection_items SET status = 'success' WHERE inspection = ? AND status = 'running'`,
		t.inspectionId,
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}
