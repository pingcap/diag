package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getSoftwareInfoHandler struct {
	m model.Model
}

func SoftwareInfo(m model.Model) http.Handler {
	return &getSoftwareInfoHandler{m}
}

func (h *getSoftwareInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionSoftwareInfo).ServeHTTP(w, r)
}

func (h *getSoftwareInfoHandler) getInspectionSoftwareInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionSoftwareInfo(inspectionId)
	if err != nil {
		log.Error("get inspection software version:", err)
		return nil, utils.DatabaseQueryError
	}

	conclusions := make([]map[string]interface{}, 0)
	for _, comp := range info {
		if comp.Version.GetTag("status") != "" {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  comp.Version.GetTag("status"),
				"message": comp.Version.GetTag("message"),
			})
		}
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       info,
	}, nil
}
