package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/checker/render"
	"github.com/pingcap/log"
	"go.uber.org/zap"
)

// unique rulename
// attribute: required_data, e.g. config.tidb, performance.dashboard
// attribute: check_type, e.g. config, performance
// -------build compute -----------
// range rule, find data tag, Assembly a compute unit, insert into [hash([]data_index)]{[]rule, data}
// -------wrapper result ------------
// type.component.rulename rulename	data + result -> [rulename][hash([]data_index)]checkinfo
// how checkinfo produce
// data.digest + rule.level + rule.variable_name + rule.variable_value

type Wrapper struct {
	SourceData     *proto.SourceDataV2
	Render         *render.ResultWrapper
	RuleResult     map[string]proto.PrintTemplate
	RuleSet        map[string]*proto.Rule
	computeUnitSet map[string]*ComputeUnit
}

func NewWrapper(sd *proto.SourceDataV2, rs map[string]*proto.Rule, rd *render.ResultWrapper) *Wrapper {
	return &Wrapper{
		SourceData:     sd,
		RuleSet:        rs,
		Render:         rd,
		RuleResult:     make(map[string]proto.PrintTemplate),
		computeUnitSet: make(map[string]*ComputeUnit),
	}
}

func (w *Wrapper) Start(ctx context.Context) error {
	for _, rule := range w.RuleSet {
		dataSet, err := w.GetDataSet(rule.NameStruct)
		if err != nil {
			return fmt.Errorf("get DataSet Faield, %s", err.Error())
		}
		for _, data := range dataSet {
			if cu, ok := w.computeUnitSet[data.UqiTag]; ok {
				cu.Rules = append(cu.Rules, rule)
			} else {
				cu := NewComputeUnit(data)
				cu.Rules = append(cu.Rules, rule)
				w.computeUnitSet[data.UqiTag] = cu
			}
		}
	}
	if err := w.Exec(); err != nil {
		return err
	}
	w.Render.Output(ctx, w.RuleResult) // todo@toto add ruleResultPrint
	return nil
}

func (w *Wrapper) Exec() error {
	for _, cu := range w.computeUnitSet {
		result, err := cu.Compute() // todo@toto add dataset
		hd := cu.GetDataSet()
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if err := w.PackageResult(hd, result); err != nil {
			log.Error(fmt.Sprintf("package result failed, %s", err.Error()))
			return err
		}
	}
	return nil
}

func (w *Wrapper) PackageResult(hd *proto.HandleData, resultset map[string]interface{}) error {
	for rulename, res := range resultset {
		rule, isExisted := w.RuleSet[rulename]
		if !isExisted {
			log.Error("no such rule")
			return fmt.Errorf("no such rule %s", rulename)
		}
		rulePrinter, ok := w.RuleResult[rulename]
		if !ok {
			switch rule.CheckType {
			case proto.ConfigType, proto.DefaultConfigType: // to move a global check type define
				rulePrinter = proto.NewConfPrintTemplate(rule) // todo@toto add new func
			case proto.PerformanceType:
				rulePrinter = proto.NewSQLPerformancePrintTemplate(rule) // todo@toto add new func
			default:
				log.Error("can't handle such type rule: ", zap.String("checktype", rule.CheckType))
				return fmt.Errorf("can't handle %s type rule: ", rule.CheckType)
			}
		}
		if err := rulePrinter.CollectResult(hd, res); err != nil {
			log.Error("collectResult failed")
			return fmt.Errorf("collectResult failed: %s", err.Error())
		}
		w.RuleResult[rulename] = rulePrinter
	}
	return nil
}

func (w *Wrapper) GetDataSet(namestructs string) ([]*proto.HandleData, error) {
	// repackage data
	// todo@toto split namestruct and fetch n * n data
	valClasses := w.SplitNamestruct(namestructs)
	chainData := make([][]proto.Data, 0)
	for _, valClass := range valClasses {
		singleclassData, err := w.FindData(valClass)
		if err != nil {
			log.Error(fmt.Sprintf("can't find data %s", err))
			return nil, err
		}
		chainData = append(chainData, singleclassData)

	}
	cd := w.CrossData(chainData)
	return w.GenHandleData(cd), nil
}

func (w *Wrapper) CrossData(oriData [][]proto.Data) [][]proto.Data {
	if len(oriData) <= 1 { // 1 * 2 -> 2 * 1
		crossData := make([][]proto.Data, 0)
		for _, d := range oriData[0] {
			crossData = append(crossData, []proto.Data{d})
		}
		return crossData
	}
	newComs := w.CrossData(oriData[1:])
	nCross := make([][]proto.Data, 0)
	for _, dgroup := range oriData[0] {
		for _, newCom := range newComs { // 1 * 2
			nRow := append(newCom, dgroup)
			nCross = append(nCross, nRow)
		}
	}
	return nCross
}

func (w *Wrapper) GenHandleData(ds [][]proto.Data) []*proto.HandleData {
	hds := make([]*proto.HandleData, 0)
	for _, d := range ds {
		hds = append(hds, proto.NewHandleData(d))
	}
	return hds
}

func (w *Wrapper) SplitNamestruct(namestructs string) []string {
	ns := strings.Split(namestructs, ",")
	return ns
}

func (w *Wrapper) FindData(namestruct string) ([]proto.Data, error) {
	match, err := regexp.MatchString("(.*)Config", namestruct)
	if err != nil {
		log.Error("regexp failed")
		return nil, err
	}
	if match {
		configData, ok := w.SourceData.NodesData[namestruct]
		if !ok {
			return nil, fmt.Errorf("no such namestruct: %s", namestruct)
		}
		reData := make([]proto.Data, 0)
		for _, d := range configData {
			reData = append(reData, d)
		}
		return reData, nil
	} else if namestruct == "performance.dashboard" {
		sqlPerformance := w.SourceData.DashboardData
		return []proto.Data{sqlPerformance}, nil
	}
	return nil, fmt.Errorf("no such namestruct: %s", namestruct)
}
