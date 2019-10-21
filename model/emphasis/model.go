package emphasis

import (
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	"time"
)

type Model interface {
	ListAllEmphasis() ([]*Emphasis, error)
	ListAllEmphasisOfInstance(instanceId string) ([]*Emphasis, error)
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

func (*emphasis) ListAllEmphasis() ([]*Emphasis, error) {
	panic("implement me")
}

func (*emphasis) ListAllEmphasisOfInstance(instanceId string) ([]*Emphasis, error) {
	panic("implement me")
}

func (*emphasis) GenerateEmphasis(InvestStart time.Time, InvestEnd time.Time, InvestProblem string) (*Emphasis, error) {
	panic("implement me")
}

func (*emphasis) GetEmphasis(uuid string) (*Emphasis, error) {
	panic("implement me")
}
