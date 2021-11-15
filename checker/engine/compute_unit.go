package engine

import (
	genginebuilder "github.com/bilibili/gengine/builder"
	genginecontext "github.com/bilibili/gengine/context"
	"github.com/bilibili/gengine/engine"
	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/pkg/utils"
	"github.com/pingcap/log"
	"go.uber.org/zap"
)

func GetRegisterFunc() map[string]interface{} {
	return map[string]interface{}{
		"ToInt":       utils.ValueToInt,
		"ToString":    utils.ValueToString,
		"ToBool":      utils.ValueToBool,
		"ToFloat":     utils.ValueToFloat,
		"FlatMap":     utils.FlatMap,
		"ElemInRange": utils.ElemInRange,
	}
}

// unique {data1, data2}
type ComputeUnit struct {
	Rules      []*proto.Rule
	HandleData *proto.HandleData
}

func NewComputeUnit(hd *proto.HandleData) *ComputeUnit {
	return &ComputeUnit{
		Rules:      make([]*proto.Rule, 0),
		HandleData: hd,
	}
}

func (u *ComputeUnit) Compute() (map[string]interface{}, error) {
	if !u.HandleData.IsValid {
		return u.MockEmptyResult(), nil
	}
	dataContext := genginecontext.NewDataContext()
	u.Register(dataContext)
	ruleBuilder := genginebuilder.NewRuleBuilder(dataContext)
	rulestr := u.RuleSplicing()
	if err := ruleBuilder.BuildRuleFromString(rulestr); err != nil {
		log.Error("build rule %+v err:%s ", zap.Any("rule", rulestr), zap.Error(err))
		return nil, err
	}
	eng := engine.NewGengine()
	if err := eng.ExecuteConcurrent(ruleBuilder); err != nil {
		log.Error("execute rule error: %v", zap.Error(err))
		return nil, err
	}
	result, err := eng.GetRulesResultMap()
	if err != nil {
		log.Error("fetch execution result failed, ", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (u *ComputeUnit) Register(dataContext *genginecontext.DataContext) {
	// register global func
	GlobalRegisterFunc := GetRegisterFunc()
	for globalName, globalFunc := range GlobalRegisterFunc {
		dataContext.Add(globalName, globalFunc)
	}
	// register data acting name
	for _, d := range u.HandleData.Data {
		dataContext.Add(d.ActingName(), d)
	}
}

func (u *ComputeUnit) RuleSplicing() string {
	rulestr := ""
	for _, rule := range u.Rules {
		rulestr += rule.ExecuteRule + "\n"
	}
	return rulestr
}

func (u *ComputeUnit) GetDataSet() *proto.HandleData {
	return u.HandleData
}

func (u *ComputeUnit) MockEmptyResult() map[string]interface{} {
	emptyRes := make(map[string]interface{})
	for _, rule := range u.Rules {
		emptyRes[rule.Name] = nil
	}
	return emptyRes
}
