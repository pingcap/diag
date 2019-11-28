package emphasis

import (
	"time"

	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
)

const (
	Success   string = "success"
	Exception        = "exception"
	Running          = "running"
)

type Emphasis struct {
	Uuid                string    `json:"uuid"`
	InstanceId          string    `json:"instance_id"`
	InstanceName        string    `json:"instance_name"`
	CreatedTime         time.Time `json:"created_time"`
	InvestgatingStart   time.Time `json:"investgating_start"`
	InvestgatingEnd     time.Time `json:"investgating_end"`
	User                string    `json:"user"`
	InvestgatingProblem string    `json:"investgating_problem"`

	Status string `json:"status"` // The status of "running" | "exception" | "success" .

	RelatedProblems []*Problem `json:"related_problems" gorm:"foreignkey:UserRefer"`
}

func (emp *Emphasis) CorrespondInspection() *inspection.Inspection {
	return &inspection.Inspection{
		Uuid:         emp.Uuid,
		InstanceId:   emp.InstanceId,
		CreateTime:   utils.FromTime(emp.CreatedTime),
		ScrapeBegin:  utils.FromTime(emp.InvestgatingStart),
		ScrapeEnd:    utils.FromTime(emp.InvestgatingEnd),
		InstanceName: emp.InstanceName,
		Status:       emp.Status,
		Type:         "emphasis",
		User:         emp.User,

		Problem:      emp.InvestgatingProblem,
	}
}

func InspectionToEmphasis(insp *inspection.Inspection) *Emphasis {
	return &Emphasis{
		Uuid:                insp.Uuid,
		InstanceId:          insp.InstanceId,
		CreatedTime:         insp.CreateTime.Time,
		InvestgatingStart:   insp.ScrapeBegin.Time,
		InvestgatingEnd:     insp.ScrapeEnd.Time,
		Status:              insp.Status,
		InvestgatingProblem: insp.Problem,
		InstanceName:        insp.InstanceName,
		User:                insp.User,
	}
}
