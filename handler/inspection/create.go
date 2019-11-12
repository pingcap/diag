package inspection

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type DiagnoseWorker interface {
	Collect(inspectionId, inspectionType string, config *model.Config) error
	Analyze(inspectionId string) error
}

type createInspectionHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
	w DiagnoseWorker
}

func CreateInspection(c *bootstrap.ForesightConfig, m model.Model, w DiagnoseWorker) http.Handler {
	return &createInspectionHandler{c, m, w}
}

func (h *createInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createInspection).ServeHTTP(w, r)
}

func (h *createInspectionHandler) createInspection(r *http.Request, c *model.Config) (*model.Inspection, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	c.InstanceId = instanceId

	inspection := &model.Inspection{
		Uuid:       inspectionId,
		InstanceId: instanceId,
		Status:     "running",
		Type:       "manual",
	}
	err := h.m.SetInspection(inspection)
	if err != nil {
		log.Error("set inspection: ", err)
		return nil, utils.DatabaseInsertError
	}

	go func() {
		if err := h.w.Collect(inspectionId, "manual", c); err != nil {
			log.Error("collect ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "collect failed"
			h.m.SetInspection(inspection)
			return
		}

		if err := h.w.Analyze(inspectionId); err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "analyze failed"
			h.m.SetInspection(inspection)
			return
		}
	}()

	return inspection, nil
}
