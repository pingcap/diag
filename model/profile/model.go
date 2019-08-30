package profile

import (
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/wrapper/db"
)

type Model interface {
	ListAllProfiles(page, size int64, profDir string) ([]*Profile, int, error)
	ListProfiles(instanceId string, page, size int64, profDir string) ([]*Profile, int, error)
	GetProfile(profId, profDir string) (*Profile, error)
}

func New(db db.DB) Model {
	return &profile{db, report.New(db)}
}

type profile struct {
	db db.DB
	r  report.Model
}

func (m *profile) ListAllProfiles(page, size int64, profDir string) ([]*Profile, int, error) {
	profs := []*Profile{}
	insps := []*inspection.Inspection{}
	query := m.db.Model(&inspection.Inspection{}).Where(PROF_FILTER).Order("create_time desc")

	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	count := 0
	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	// transform inspection to profile
	for _, insp := range insps {
		prof := fromInspection(insp, profDir)
		if symptoms, err := m.r.GetInspectionSymptoms(insp.Uuid); err != nil {
			return nil, 0, err
		} else if len(symptoms) != 0 {
			prof.Status = "exception"
			prof.Message = "collect failed"
			profs = append(profs, prof)
		} else {
			profs = append(profs, prof)
		}
	}

	return profs, count, nil
}

func (m *profile) ListProfiles(instanceId string, page, size int64, profDir string) ([]*Profile, int, error) {
	profs := []*Profile{}
	insps := []*inspection.Inspection{}
	query := m.db.Model(&inspection.Inspection{}).
		Where(PROF_FILTER).
		Where(&inspection.Inspection{InstanceId: instanceId}).
		Order("create_time desc")

	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	count := 0
	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	// transform inspection to profile
	for _, insp := range insps {
		if symptoms, err := m.r.GetInspectionSymptoms(insp.Uuid); err != nil {
			return nil, 0, err
		} else if len(symptoms) != 0 {
			insp.Status = "exception"
			insp.Message = "collect failed"
		}
		profs = append(profs, fromInspection(insp, profDir))
	}

	return profs, count, nil
}

func (m *profile) GetProfile(profId, profDir string) (*Profile, error) {
	insp := inspection.Inspection{}

	if err := m.db.Where(&inspection.Inspection{Uuid: profId}).Take(&insp).Error(); err != nil {
		return nil, err
	}

	return fromInspection(&insp, profDir), nil
}
