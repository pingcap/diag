package report

import (
	"reflect"
	"runtime"

	"github.com/pingcap/tidb-foresight/wraper/db"
	log "github.com/sirupsen/logrus"
)

type Report struct {
	db           db.DB
	inspectionId string
	Items        interface{} `json:"items,omitempty"`
	Symptoms     interface{} `json:"symptoms,omitempty"`
	BasicInfo    interface{} `json:"basic,omitempty"`
	DBInfo       interface{} `json:"dbinfo,omitempty"`
	AlertInfo    interface{} `json:"alert,omitempty"`
	ResourceInfo interface{} `json:"resource,omitempty"`
	SlowLogInfo  interface{} `json:"slow_log,omitempty"`
	HardwareInfo interface{} `json:"hardware,omitempty"`
	SoftwareInfo interface{} `json:"software_version,omitempty"`
	ConfigInfo   interface{} `json:"software_config,omitempty"`
	NetworkInfo  interface{} `json:"network,omitempty"`
	DemsgLog     interface{} `json:"demsg,omitempty"`
}

func NewReport(db db.DB, inspectionId string) *Report {
	return &Report{
		db:           db,
		inspectionId: inspectionId,
	}
}

func runAll(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			fname := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
			log.Error(fname, ":", err)
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
		r.loadSlowLogInfo,
		r.loadHardwareInfo,
		r.loadNetworkInfo,
		r.loadDemsgLog,
		r.loadSoftwareInfo,
		r.loadConfigInfo,
	)
}
