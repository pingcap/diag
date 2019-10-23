package emphasis

import "time"

type Emphasis struct {
	Uuid              string    `json:"uuid"`
	InstanceId        string    `json:"instance_id"`
	CreatedTime       time.Time `json:"created_time"`
	InvestgatingStart time.Time `json:"investgating_start"`
	InvestgatingEnd   time.Time `json:"investgating_end"`

	InvestgatingProblem string `json:"investgating_problem"`

	RelatedProblems []Emphasis `json:"related_problems" gorm:"foreignkey:UserRefer"`
}
