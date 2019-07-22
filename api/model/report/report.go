package report

import (
	"database/sql"
)

type Report struct {
	db           *sql.DB
	inspectionId string
	Items        interface{} `json:"items,omitempty"`
	BasicInfo    interface{} `json:"basic,omitempty"`
	DBInfo       interface{} `json:"dbinfo,omitempty"`
}

func NewReport(db *sql.DB, inspectionId string) *Report {
	return &Report{
		db:           db,
		inspectionId: inspectionId,
	}
}

func runAll(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (r *Report) Load() error {
	// TODO: add more info
	return runAll(
		r.loadItems,
		r.loadBasicInfo,
		r.loadDBInfo,
	)
}
