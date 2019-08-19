package report

import (
	"time"
)

type BasicInfo struct {
	InspectionId      string    `json:"-"`
	ClusterName       string    `json:"cluster_name"`
	ClusterCreateTime time.Time `json:"cluster_create_time"`
	InspectTime       time.Time `json:"inspect_time"`
	TidbAlive         int       `json:"tidb_alive"`
	TikvAlive         int       `json:"tikv_alive"`
	PdAlive           int       `json:"pd_alive"`
}

func (m *report) GetInspectionBasicInfo(inspectionId string) (*BasicInfo, error) {
	basic := BasicInfo{}

	if err := m.db.Where(&BasicInfo{InspectionId: inspectionId}).Take(&basic).Error(); err != nil {
		return nil, err
	}

	return &basic, nil
}

func (m *report) ClearInspectionBasicInfo(inspectionId string) error {
	return m.db.Delete(&BasicInfo{InspectionId: inspectionId}).Error()
}

func (m *report) InsertInspectionBasicInfo(info *BasicInfo) error {
	return m.db.Create(info).Error()
}
