package inspection

import (
	"net/http"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type importInspectionHandler struct {
	c *bootstrap.ForesightConfig
	m InspectionImportor
}

func ImportInspection(c *bootstrap.ForesightConfig, m InspectionImportor) http.Handler {
	return &importInspectionHandler{c, m}
}

func (h *importInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.importInspection).ServeHTTP(w, r)
}

func (h *importInspectionHandler) importInspection(r *http.Request) (*model.Inspection, utils.StatusError) {
	inspectionId, err := utils.UploadInspection(h.c.Home, r)
	if err != nil {
		return nil, err
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

	go func() {
		if err := utils.Analyze(h.c.Analyzer, h.c.Home, h.c.Influx.Endpoint, h.c.Prometheus.Endpoint, inspectionId); err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "analyze failed"
			h.m.SetInspection(inspection)
			return
		}
	}()

	return inspection, nil
}
