package inspection

import (
	"net/http"

	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
)

type InspectionDeletor interface {
	DeleteInspection(inspectionId string) error
}

type InspectionGeter interface {
	GetInspection(inspectionId string) (*model.Inspection, error)
}

type InspectionLister interface {
	ListInspections(instanceId string, page, size int64) ([]*model.Inspection, int, error)
}

type AllInspectionLister interface {
	ListAllInspections(page, size int64) ([]*model.Inspection, int, error)
}

type InspectionSeter interface {
	SetInspection(inspection *model.Inspection) error
}

type InspectionUploader interface {
	UploadInspection(home string, r *http.Request) (string, utils.StatusError)
}

type InspectionImportor interface {
	InspectionSeter
}

type InspectionCreator interface {
	InspectionSeter
	GetInstance(instanceId string) (*model.Instance, error)
}
