package emphasis

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	"time"
)

type Model interface {
	ListAllEmphasis(page, size int64) ([]*Emphasis, error)
	ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, error)
	GenerateEmphasis(InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error)
	GetEmphasis(uuid string) (*Emphasis, error)
}

func New(db db.DB) Model {
	utils.MustInitSchema(db, &Emphasis{})
	return &emphasis{db}
}

type emphasis struct {
	db db.DB
}

func (e *emphasis) ListAllEmphasis(page, size int64) ([]*Emphasis, int, error) {
	insps := []*Emphasis{}
	count := 0
	query := e.db.Model(&Emphasis{}).Order("created_time desc")

	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	return insps, count, nil
}

func (*emphasis) ListAllEmphasisOfInstance(page, size int64, instanceId string) ([]*Emphasis, error) {
	panic("implement me")
}

func (*emphasis) GenerateEmphasis(InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error) {
	panic("implement me")
}

func (*emphasis) GetEmphasis(uuid string) (*Emphasis, error) {
	panic("implement me")
}
