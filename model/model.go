package model

import (
	"github.com/pingcap/tidb-foresight/model/config"
	"github.com/pingcap/tidb-foresight/model/emphasis"
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/model/instance"
	"github.com/pingcap/tidb-foresight/model/logs"
	"github.com/pingcap/tidb-foresight/model/profile"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/model/statement"
	"github.com/pingcap/tidb-foresight/wrapper/db"
)

type ReportModel = report.Model
type ConfigModel = config.Model
type InspectionModel = inspection.Model
type InstanceModel = instance.Model
type ProfileModel = profile.Model
type LogModel = logs.Model
type EmphasisModel = emphasis.Model
type StatementModel = statement.Model

type Model interface {
	ReportModel
	ConfigModel
	InspectionModel
	InstanceModel
	ProfileModel
	LogModel
	EmphasisModel
	StatementModel
}

type model struct {
	ReportModel
	ConfigModel
	InspectionModel
	InstanceModel
	ProfileModel
	LogModel
	EmphasisModel
	StatementModel
}

func New(db db.DB) Model {
	return &model{
		report.New(db),
		config.New(db),
		inspection.New(db),
		instance.New(db),
		profile.New(db),
		logs.New(db),
		emphasis.New(db),
		statement.New(db),
	}
}
