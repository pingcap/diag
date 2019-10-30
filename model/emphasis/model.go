package emphasis

import (
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	"time"
)

type Model interface {
	ListAllEmphasis(page, size int64) ([]*Emphasis, int, error)
	ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, int, error)
	CreateEmphasis(*Emphasis) error
	//GenerateEmphasis(InstanceId string, InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error)
	GetEmphasis(uuid string) (*Emphasis, error)
	// Add Problem for Emphasis
	AddProblem(inspectionId string, problem *Problem) error
	// Load all problems for Emphasis
	LoadAllProblems(emp *Emphasis) ([]*Problem, error)
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Emphasis{})
	return &emphasis{db}
}

type emphasis struct {
	db db.DB
}

func (e *emphasis) AddProblem(inspectionId string, problem *Problem) error {
	problem.InspectionId = inspectionId
	problem.CreateTime = utils.FromTime(time.Now())
	err := e.db.Create(problem).Error()
	return err
}

func (e *emphasis) LoadAllProblems(emp *Emphasis) ([]*Problem, error) {
	problems := []*Problem{}
	err := e.db.Model(&Problem{}).Where("instance_id = ?", emp.Uuid).Order("create_time desc").Find(&problems).Error()
	return problems, err
}

func (e *emphasis) CreateEmphasis(emp *Emphasis) error {
	return e.db.Create(emp.CorrespondInspection()).Error()
}

// The helper function for paging.
func (e *emphasis) paging(query db.DB, page, size int64) ([]*Emphasis, int, error) {
	insps := []*inspection.Inspection{}
	count := 0
	//query = query.Where("type = ?", "emphasis")
	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}
	emps := make([]*Emphasis, 0)
	for _, v := range insps {
		emps = append(emps, InspectionToEmphasis(v))
	}
	return emps, count, nil
}

func (e *emphasis) ListAllEmphasis(page, size int64) ([]*Emphasis, int, error) {
	query := e.db.Model(&inspection.Inspection{}).Order("create_time desc")
	return e.paging(query, page, size)
}

func (e *emphasis) ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, int, error) {
	query := e.db.Model(&inspection.Inspection{}).Where("instance_id = ?", instanceId).Order("create_time desc")
	return e.paging(query, page, size)
}

func (e *emphasis) GetEmphasis(uuid string) (*Emphasis, error) {
	emph := Emphasis{}

	if err := e.db.Where("uuid = ?", uuid).Take(&emph).Error(); err != nil {
		return nil, err
	}
	return &emph, nil
}
