package report

import (
	ts "github.com/pingcap/tidb-foresight/utils/tagd-value/string"
)

type TopologyInfo struct {
	InspectionId string    `json:"-"`
	Name         string    `json:"name"`
	NodeIp       string    `json:"node_ip"`
	Port         string    `json:"port"`
	Status       ts.String `json:"status"`
}

func (m *report) GetInspectionTopologyInfo(inspectionId string) ([]*TopologyInfo, error) {
	infos := []*TopologyInfo{}

	if err := m.db.Where(&TopologyInfo{InspectionId: inspectionId}).Find(&infos).Error(); err != nil {
		return nil, err
	}

	return infos, nil
}

func (m *report) ClearInspectionTopologyInfo(inspectionId string) error {
	return m.db.Delete(&TopologyInfo{}, "inspection_id = ?", inspectionId).Error()
}

func (m *report) InsertInspectionTopologyInfo(topo *TopologyInfo) error {
	return m.db.Create(topo).Error()
}
