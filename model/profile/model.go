package profile

import (
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/wraper/db"
)

type Model interface {
	ListAllProfiles(page, size int64, profDir string) ([]*Profile, int, error)
	ListProfiles(instanceId string, page, size int64, profDir string) ([]*Profile, int, error)
	GetProfile(profId, profDir string) (*Profile, error)
}

func New(db db.DB) Model {
	return &profile{db}
}

type profile struct {
	db db.DB
}

func (m *profile) ListAllProfiles(page, size int64, profDir string) ([]*Profile, int, error) {
	profs := []*Profile{}
	insps := []*inspection.Inspection{}
	query := m.db.Model(&inspection.Inspection{}).Where(PROF_FILTER)

	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	count := 0
	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	// transform inspection to profile
	for _, insp := range insps {
		if prof, err := fromInspection(insp, profDir); err != nil {
			return nil, 0, err
		} else {
			profs = append(profs, prof)
		}
	}

	return profs, count, nil
}

func (m *profile) ListProfiles(instanceId string, page, size int64, profDir string) ([]*Profile, int, error) {
	profs := []*Profile{}
	insps := []*inspection.Inspection{}
	query := m.db.Model(&inspection.Inspection{}).Where(PROF_FILTER).Where(&inspection.Inspection{InstanceId: instanceId})

	if err := query.Offset((page - 1) * size).Limit(size).Find(&insps).Error(); err != nil {
		return nil, 0, err
	}

	count := 0
	if err := query.Count(&count).Error(); err != nil {
		return nil, 0, err
	}

	// transform inspection to profile
	for _, insp := range insps {
		if prof, err := fromInspection(insp, profDir); err != nil {
			return nil, 0, err
		} else {
			profs = append(profs, prof)
		}
	}

	return profs, count, nil
}

func (m *profile) GetProfile(profId, profDir string) (*Profile, error) {
	insp := inspection.Inspection{}

	if err := m.db.Where(&inspection.Inspection{Uuid: profId}).Take(&insp).Error(); err != nil {
		return nil, err
	}

	return fromInspection(&insp, profDir)
}
