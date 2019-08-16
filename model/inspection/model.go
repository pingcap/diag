package inspection

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model interface {
	ListAllInspections(page, size int64) ([]*Inspection, int, error)
	ListInspections(instId string, page, size int64) ([]*Inspection, int, error)
	SetInspection(insp *Inspection) error
	GetInspection(inspId string) (*Inspection, error)
	DeleteInspection(inspId string) error
	UpdateInspectionStatus(inspId, status string) error
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Inspection{})
	return &inspection{db}
}

type inspection struct {
	db db.DB
}
