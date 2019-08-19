package logs

import (
	"net/http"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type importLogHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func ImportLog(c *bootstrap.ForesightConfig, m model.Model) http.Handler {
	return &importLogHandler{c, m}
}

func (h *importLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.importLog).ServeHTTP(w, r)
}

func (h *importLogHandler) importLog(r *http.Request) (*model.LogEntity, utils.StatusError) {
	inspectionId, err := utils.UploadInspection(h.c.Home, r)
	if err != nil {
		log.Error("upload inspection:", err)
		return nil, utils.FileOpError
	}

	if err := utils.UnpackInspection(h.c.Home, inspectionId); err != nil {
		log.Error("unpack: ", err)
		return nil, utils.FileOpError
	}

	inspection := &model.Inspection{
		Uuid:   inspectionId,
		Status: "running",
	}
	if err := h.m.SetInspection(inspection); err != nil {
		log.Error("create inspection: ", err)
		return nil, utils.DatabaseInsertError
	}

	if err := utils.Analyze(h.c.Analyzer, h.c.Home, h.c.Influx.Endpoint, h.c.Prometheus.Endpoint, inspectionId); err != nil {
		log.Error("analyze ", inspectionId, ": ", err)
		inspection.Status = "exception"
		inspection.Message = "analyze failed"
		h.m.SetInspection(inspection)
		return nil, utils.SystemOpError
	}

	if inspection, err := h.m.GetInspection(inspectionId); err != nil {
		log.Error("get inspection detail:", err)
		return nil, utils.DatabaseQueryError
	} else {
		return &model.LogEntity{Id: inspection.Uuid, InstanceName: inspection.InstanceName}, nil
	}
}
