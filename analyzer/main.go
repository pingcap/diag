package main

import (
	"database/sql"
	"flag"
	"fmt"
	"path"
	"reflect"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pingcap/tidb-foresight/analyzer/task"
	log "github.com/sirupsen/logrus"
)

type Analyzer struct {
	inspectionId string
	home         string
	db           *sql.DB
	data         task.TaskData
}

func NewAnalyzer(home, inspectionId string) (*Analyzer, error) {
	analyzer := &Analyzer{
		inspectionId: inspectionId,
		home:         home,
	}
	var err error

	analyzer.db, err = sql.Open("sqlite3", path.Join(home, "sqlite.db"))
	if err != nil {
		return nil, err
	}

	return analyzer, nil
}

func (a *Analyzer) runTasks(tasks ...func(task.BaseTask) task.Task) error {
	base := task.NewTask(a.inspectionId, a.home, &a.data, a.db)
	for _, t := range tasks {
		if err := t(base).Run(); err != nil {
			fname := runtime.FuncForPC(reflect.ValueOf(t).Pointer()).Name()
			log.Error("run task ", fname, " :", err)
			err := base.InsertSymptom(
				"exception",
				fmt.Sprintf("error on running analyze task: %s", fname),
				"this error is not about the tidb cluster you are running, it's about tidb-foresight itself",
			)
			return err
		}
	}
	return nil
}

func (a *Analyzer) Run() error {
	err := a.runTasks(
		task.Clear,

		// parse stage
		task.ParseArgs,
		task.ParseStatus,
		task.ParseTopology,
		task.ParseMetric,
		task.SaveMetric, // the matric should be save since bellow parse depend on it
		task.ParseMeta,
		task.ParseDBInfo,
		task.ParseAlert,
		task.ParseInsight,
		task.ParseResource,

		// save stage
		task.SaveItems,
		task.SaveInspection,
		task.SaveBasicInfo,
		task.SaveDBInfo,
		task.SaveSlowLogInfo,
		task.SaveNetwork,
		task.SaveAlert,
		task.SaveHardwareInfo,
		task.SaveDmesg,
    task.SaveLog,
		task.SaveProfile,
		task.SaveResource,
		task.SaveSoftwareVersion,
		task.SaveSoftwareConfig,

		// analyze stage
		task.Analyze,
	)

	if err == nil {
		_, err = a.db.Exec(
			"UPDATE inspections SET status = ? WHERE id = ?",
			"success", a.inspectionId,
		)
		return err
	} else {
		_, err = a.db.Exec(
			"UPDATE inspections SET status = ?, message = ? WHERE id = ?",
			"exception", "error on running analyzer", a.inspectionId,
		)
		return err
	}

	return nil
}

func main() {
	home := flag.String("home", "/tmp/tidb-foresight", "the tidb-foresight data directory")
	inspectionId := flag.String("inspection-id", "", "the inspection to be analyze")
	flag.Parse()

	if *inspectionId == "" {
		log.Panic("the inspection-id must be specified")
	}

	analyzer, err := NewAnalyzer(*home, *inspectionId)
	if err != nil {
		log.Panic("init analyzer: ", err)
	}

	err = analyzer.Run()
	if err != nil {
		log.Error("run analyzer: ", err)
	}
}
