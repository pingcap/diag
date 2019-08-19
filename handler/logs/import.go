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
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on upload file")
	}

	if err := utils.UnpackInspection(h.c.Home, inspectionId); err != nil {
		log.Error("unpack: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on unpack file")
	}

	inspection := &model.Inspection{
		Uuid:   inspectionId,
		Status: "running",
	}
	if err := h.m.SetInspection(inspection); err != nil {
		log.Error("create inspection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	if err := utils.Analyze(h.c.Analyzer, h.c.Home, h.c.Influx.Endpoint, h.c.Prometheus.Endpoint, inspectionId); err != nil {
		log.Error("analyze ", inspectionId, ": ", err)
		inspection.Status = "exception"
		inspection.Message = "analyze failed"
		h.m.SetInspection(inspection)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on import log")
	}

	if inspection, err := h.m.GetInspection(inspectionId); err != nil {
		log.Error("get inspection detail:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	} else if inspection == nil {
		log.Error("not found inspection after import log")
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "inspection not found")
	} else {
		return &model.LogEntity{Id: inspection.Uuid, InstanceName: inspection.InstanceName}, nil
	}
}
