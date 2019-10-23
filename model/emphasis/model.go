package emphasis

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
)

type Model interface {
	ListAllEmphasis(page, size int64) ([]*Emphasis, int, error)
	ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, int, error)
	CreateEmphasis(*Emphasis) error
	//GenerateEmphasis(InstanceId string, InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error)
	GetEmphasis(uuid string) (*Emphasis, error)
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Emphasis{})
	return &emphasis{db}
}

type emphasis struct {
	db db.DB
}

func (e *emphasis) CreateEmphasis(emp *Emphasis) error {
	return e.db.Create(emp).Error()
}

// 分页的辅助函数
func (e *emphasis) paging(query db.DB, page, size int64) ([]*Emphasis, int, error) {
	insps := []*Emphasis{}
	count := 0
	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	return insps, count, nil
}

func (e *emphasis) ListAllEmphasis(page, size int64) ([]*Emphasis, int, error) {
	query := e.db.Model(&Emphasis{}).Order("created_time desc")
	return e.paging(query, page, size)
}

func (e *emphasis) ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, int, error) {
	query := e.db.Model(&Emphasis{}).Where("instance_id = ?", instanceId).Order("created_time desc")
	return e.paging(query, page, size)
}

//func (e *emphasis) GenerateEmphasis(InstanceId string, InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error) {
//	emph := Emphasis{InstanceId: InstanceId, InvestgatingStart: InvestStart, InvestgatingEnd: InvestEnd, InvestgatingProblem: InvestProblem}
//
//	if err := e.db.Create(&emph).Error(); err != nil {
//		return nil, err
//	}
//	return &emph, nil
//}

func (e *emphasis) GetEmphasis(uuid string) (*Emphasis, error) {
	emph := Emphasis{}

	if err := e.db.Where(&Emphasis{Uuid: uuid}).Take(&emph).Error(); err != nil {
		return nil, err
	}
	return &emph, nil
}
