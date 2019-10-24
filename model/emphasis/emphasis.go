package emphasis

import (
	"github.com/pingcap/tidb-foresight/model/inspection"
	"time"
)

type Emphasis struct {
	Uuid              string    `json:"uuid"`
	InstanceId        string    `json:"instance_id"`
	CreatedTime       time.Time `json:"created_time"`
	InvestgatingStart time.Time `json:"investgating_start"`
	InvestgatingEnd   time.Time `json:"investgating_end"`

	InvestgatingProblem string `json:"investgating_problem"`

	RelatedProblems []Emphasis `json:"related_problems" gorm:"foreignkey:UserRefer"`
}

// TODO: 类型转换
func (emp *Emphasis) CorrespondInspection() inspection.Inspection {
	panic("implement me")
}

// TODO:
func InspectionToEmphasis(insp *inspection.Inspection) Emphasis {
	panic("implement me")
}
