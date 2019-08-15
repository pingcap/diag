package profile

import (
	"github.com/pingcap/tidb-foresight/model"
)

type ProfileLister interface {
	ListProfiles(instanceId string, page, size int64, profileDir string) ([]*model.Profile, int, error)
}

type AllProfileLister interface {
	ListAllProfiles(page, size int64, profileDir string) ([]*model.Profile, int, error)
}

type ProfileCreator interface {
	SetInspection(inspection *model.Inspection) error
	GetInstance(instanceId string) (*model.Instance, error)
}

type ProfileGeter interface {
	GetProfile(profileId, profileDir string) (*model.Profile, error)
}
