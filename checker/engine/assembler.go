package engine

import (
	"fmt"
	"regexp"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/checker/render"
	"github.com/pingcap/log"
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
	CheckType      int // 1, 1<<1
	Render         *render.ResultWrapper
	RuleResult     map[string]proto.PrintTemplate
	RuleSet        map[string]*proto.Rule
	computeUnitSet map[string]*ComputeUnit
}

func NewWrapper(sd *proto.SourceDataV2, rs map[string]*proto.Rule) *Wrapper {
	return &Wrapper{
		SourceData:     sd,
		RuleSet:        rs,
		Render:         render.NewResultWrapper(rs),
		RuleResult:     make(map[string]proto.PrintTemplate),
		computeUnitSet: make(map[string]*ComputeUnit),
	}
}

func (w *Wrapper) Start() error {
	for _, rule := range w.RuleSet {
		if rule.CheckType&w.CheckType == 0 {
			continue
		}
		dataSet, err := w.GetDataSet(rule.NameStruct, w.SourceData)
		if err != nil {
			return fmt.Errorf("Get DataSet Faield, %s", err)
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
	w.Render.Output(w.RuleResult) // todo@toto add ruleResultPrint
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
		w.PackageResult(hd, result)
	}
	return nil
}

func (w *Wrapper) PackageResult(hd *proto.HandleData, resultset map[string]interface{}) error {
	for rulename, res := range resultset {
		rule, _ := w.RuleSet[rulename]
		rulePrinter, ok := w.RuleResult[rulename]
		if ok {
			rulePrinter.CollectResult(hd, res)
		} else {
			switch rule.CheckType {
			case 1: // to move a global check type define
				rulePrinter = proto.NewConfPrintTemplate(rule) // todo@toto add new func
			case 1 << 1:
				rulePrinter = proto.NewSqlPerformancePrintTemplate(rule) // todo@toto add new func
			}
		}
		if err := rulePrinter.CollectResult(hd, res); err != nil {
			log.Error("collectResult failed")
			return fmt.Errorf("collectResult failed: ", err.Error())
		}
		w.RuleResult[rulename] = rulePrinter
	}
	return nil
}

func (w *Wrapper) GetDataSet(namestruct string, sd *proto.SourceDataV2) ([]*proto.HandleData, error) {
	// repackage data
	match, err := regexp.MatchString("(.*)Config", namestruct)
	if err != nil {
		log.Error("regexp failed")
		return nil, err
	}
	if match {
		configData, ok := sd.NodesData[namestruct]
		if !ok {
			return nil, fmt.Errorf("no such namestruct: %s", namestruct)
		}
		// todo@toto slice
		dataset := make([]*proto.HandleData, 0)
		for _, conf := range configData {
			uqiTag := fmt.Sprintf("%s_%s:%d", conf.GetComponent(), conf.GetHost(), conf.GetPort())
			handledata := &proto.HandleData{
				UqiTag: uqiTag,
				Data:   []proto.Data{conf},
			}
			dataset = append(dataset, handledata)
		}
		return dataset, nil
	} else if namestruct == "performance.dashboard" {
		sqlPerformance := sd.DashboardData
		handleData := &proto.HandleData{
			UqiTag: namestruct,
			Data:   []proto.Data{sqlPerformance},
		}
		return []*proto.HandleData{handleData}, nil
	}
	return nil, fmt.Errorf("no such namestruct: %s", namestruct)
}

func (w *Wrapper) FilterRule(ruleType int) bool {
	if w.CheckType&ruleType == 0 {
		return false
	}
	return true
}