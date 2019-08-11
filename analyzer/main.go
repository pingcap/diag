package main

import (
	"flag"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	"github.com/pingcap/tidb-foresight/analyzer/analyze"
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/clear"
	"github.com/pingcap/tidb-foresight/analyzer/input"
	"github.com/pingcap/tidb-foresight/analyzer/manager"
	"github.com/pingcap/tidb-foresight/analyzer/output"
)

type Analyzer struct {
	manager *manager.TaskManager
}

func NewAnalyzer(home, inspectionId string) *Analyzer {
	analyzer := &Analyzer{
		manager: manager.NewTaskManager(),
	}

	analyzer.manager.Register(boot.Bootstrap(inspectionId, home))
	analyzer.manager.Register(clear.ClearHistory())
	analyzer.manager.Register(input.Tasks()...)
	analyzer.manager.Register(output.Tasks()...)
	analyzer.manager.Register(analyze.Tasks()...)

	return analyzer
}

func (a *Analyzer) Run() {
	a.manager.Run()
}

func main() {
	home := flag.String("home", "/tmp/tidb-foresight", "the tidb-foresight data directory")
	inspectionId := flag.String("inspection-id", "", "the inspection to be analyze")
	flag.Parse()

	if *inspectionId == "" {
		log.Panic("the inspection-id must be specified")
	}

	analyzer := NewAnalyzer(*home, *inspectionId)
	analyzer.Run()
}
