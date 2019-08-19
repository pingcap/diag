package logs

import (
	"time"

	"github.com/pingcap/tidb-foresight/log/parser"
	"github.com/pingcap/tidb-foresight/model"
)

type LogSearcher interface {
	Search(dir string, begin, end time.Time, level, text, token string) (parser.Iterator, string, error)
}

type LogImportor interface {
	GetInspection(inspectionId string) (*model.Inspection, error)
	SetInspection(inspection *model.Inspection) error
}

type LogItem struct {
	Host      string     `json:"ip"`
	Port      string     `json:"port"`
	Component string     `json:"component"`
	File      string     `json:"file"`
	Time      *time.Time `json:"time"`
	Level     string     `json:"level"`
	Content   string     `json:"content"`
}

type LogResult struct {
	Token string     `json:"token"`
	Logs  []*LogItem `json:"logs"`
}
