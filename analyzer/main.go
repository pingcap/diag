package main

import (
	"flag"
    "database/sql"
    "path"
    "github.com/pingcap/tidb-foresight/analyzer/task"
    _ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
)

type Analyzer struct {
    inspectionId string
    home string
    db *sql.DB
    data task.TaskData
}

func NewAnalyzer(home, inspectionId string) (*Analyzer, error) {
    analyzer := &Analyzer{
        inspectionId: inspectionId,
        home: home,
    }
    var err error

    analyzer.db, err = sql.Open("sqlite3", path.Join(home, "sqlite.db"))
    if err != nil {
        return nil, err
    }

    return analyzer, nil
}

func (a *Analyzer) runTasks(tasks ...func(task.BaseTask) task.Task) error {
    for _, t := range tasks {
        if err := t(task.NewTask(a.inspectionId, a.home, &a.data, a.db)).Run(); err != nil {
            return err
        }
    }
    return nil
}

func (a *Analyzer) Run() error {
    return a.runTasks(
       task.Clear,

       // parse stage
       task.ParseCollect,
       task.ParseStatus,
       task.ParseTopology,
       task.ParseMetric,
       task.SaveMetric,      // the matric should be save since bellow parse depend on it
       task.ParseMeta,
       task.ParseDBInfo,
       task.ParseAlert,
       task.ParseInsight,

       // save stage
       task.SaveItems,
       task.SaveInspection,
       task.SaveBasicInfo,
       task.SaveDBInfo,
       task.SaveSlowLogInfo,
       task.SaveAlert,
       task.SaveHardwareInfo,
       task.SaveDmesg,
       task.SaveProfile,

       // analyze stage
       task.Analyze,
    )
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
