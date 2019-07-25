package task

import (
	"database/sql"
)

type TaskData struct {
	collect  map[string]bool
	topology Topology
	status   ItemStatus
	meta     Meta
	resource Resource
	dbinfo   DBInfo
	matrix   Matrix
}

type Task interface {
	Run() error
}

type BaseTask struct {
	inspectionId string
	src          string
	data         *TaskData
	db           *sql.DB
}
