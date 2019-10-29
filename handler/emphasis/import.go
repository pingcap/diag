package emphasis

import (
	"net/http"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type ImportWorker interface {
	Analyze(inspectionId string) error
}

type importEmphasisHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
	w ImportWorker
}

func ImportInspection(c *bootstrap.ForesightConfig, m model.Model, w ImportWorker) http.Handler {
	return &importEmphasisHandler{c, m, w}
}

func (h *importEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.importEmphasis).ServeHTTP(w, r)
}

func (h *importEmphasisHandler) importEmphasis(r *http.Request) (*model.Inspection, utils.StatusError) {
	inspectionId, err := utils.UploadInspection(h.c.Home, r)
	if err != nil {
		return nil, err
	}

	if err := utils.UnpackInspection(h.c.Home, inspectionId); err != nil {
		log.Error("unpack: ", err)
		return nil, utils.FileOpError
	}

	inspection := &model.Inspection{
		Uuid:   inspectionId,
		Status: "running",
		Type:   "emphasis",
	}
	if err := h.m.SetInspection(inspection); err != nil {
		log.Error("create inspection: ", err)
		return nil, utils.DatabaseInsertError
	}

	go func() {
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
