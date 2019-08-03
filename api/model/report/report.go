package report

import (
	"database/sql"
)

type Report struct {
	db           *sql.DB
	inspectionId string
	Items        interface{} `json:"items,omitempty"`
	Symptoms     interface{} `json:"symptoms,omitempty"`
	BasicInfo    interface{} `json:"basic,omitempty"`
	DBInfo       interface{} `json:"dbinfo,omitempty"`
	AlertInfo    interface{} `json:"alert,omitempty"`
	ResourceInfo interface{} `json:"resource,omitempty"`
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
		r.loadSymptoms,
		r.loadBasicInfo,
		r.loadDBInfo,
		r.loadAlertInfo,
		r.loadResourceInfo,
	)
}
