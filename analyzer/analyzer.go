package analyzer

import (
	"runtime"

	_ "github.com/mattn/go-sqlite3"
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
		manager: manager.New(),
	}

	// Take nothing an return (*Config, *Model). Config is config for local storage.
	analyzer.manager.Register(boot.Bootstrap(inspectionId, home))
	// The method is idempotent, so we should clear all history the analyzer may create.
	analyzer.manager.Register(clear.ClearHistory())
	analyzer.manager.RunCurrentBatch()
	analyzer.manager.Register(input.Tasks()...)
	analyzer.manager.Register(output.Tasks()...)
	analyzer.manager.Register(analyze.Tasks()...)
	return analyzer
}

func (a *Analyzer) Run() {
	a.manager.ConcurrencyBatchRun(runtime.NumCPU())
}
