package statement

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	"github.com/pingcap/tidb-foresight/model/instance"
)

type Model interface {
	// Enable statements by set TiDB global variable
	EnableStatement(instanceId string) error

	// Disable statements by set TiDB global variable
	DisableStatement(instanceId string) error

	// Check if the statement is enabled in target cluster.
	// The instanceId is the instance id of target cluster.
	IsStatementEnabled(instanceId string) (bool, error)

	// Set how long the statements table should be dump to history.
	SetRefreshInterval(instanceId string, interval int) error

	// How long will TiDB refresh statements table in seconds.
	GetRefreshInterval(instanceId string) (int, error)

	// Remove outdate statements data
	StatementGc(instanceId string) error

	// Sync statements data from TiDB cluster
	StatementSync(instanceId string) error

	// Get all schemas from db
	ListSchemas(instanceId string) ([]string, error)
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Statement{})
	return &statement{db, instance.New(db)}
}

type statement struct {
	db db.DB
	inst instance.Model
}