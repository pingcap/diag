package task

import (
	"database/sql"
	log "github.com/sirupsen/logrus"
)


type AnalyzeTask struct {
	BaseTask
}

func Analyze(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &AnalyzeTask {BaseTask{inspectionId, src, data, db}}
}

// TODO
func (t *AnalyzeTask) Run() error {
	if _, err := t.db.Exec(
		`UPDATE inspection_items SET status = 'success' WHERE inspection = ? AND status = 'pending'`,
		t.inspectionId,
	); err != nil {
		log.Error("db.Exec: ", err)
		return err
	}

	return nil
}