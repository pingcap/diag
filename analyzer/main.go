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
    db *sql.DB
    src string
    data task.TaskData
}

func NewAnalyzer(src string, db string) (*Analyzer, error) {
    analyzer := &Analyzer{
        inspectionId: path.Base(src),
        src: src,
    }
    var err error

    analyzer.db, err = sql.Open("sqlite3", db)
    if err != nil {
        return nil, err
    }

    return analyzer, nil
}

func (a *Analyzer) runTasks(tasks ...func(inspectionId string, src string, data *task.TaskData, db *sql.DB) task.Task) error {
    for _, task := range tasks {
        if err := task(a.inspectionId, a.src, &a.data, a.db).Run(); err != nil {
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
        //task.SaveMetric,      // the matric should be save since bellow parse depend on it
        task.ParseMeta,
        task.ParseDBInfo,
        task.ParseResource,

        // save stage
        task.SaveItems,
        task.SaveInspection,
        task.SaveBasicInfo,
        task.SaveDBInfo,

        // analyze stage
        //task.Analyze,
    )

    return nil
}

func main() {
	src := flag.String("src", "", "the target to analyze")
	db := flag.String("db", "", "the sqlite file")

	flag.Parse()

	if *src == "" || *db == "" {
		log.Panic("both src and db must be specified")
    }

    analyzer, err := NewAnalyzer(*src, *db)
    if err != nil {
        log.Panic("init analyzer: ", err)
    }

    err = analyzer.Run()
    if err != nil {
        log.Error("run analyzer: ", err)
    }
}
