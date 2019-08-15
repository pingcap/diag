package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getSoftwareInfoHandler struct {
	m SoftwareInfoGeter
}

func SoftwareInfo(m SoftwareInfoGeter) http.Handler {
	return &getSoftwareInfoHandler{m}
}

func (h *getSoftwareInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionSoftwareInfo).ServeHTTP(w, r)
}

func (h *getSoftwareInfoHandler) getInspectionSoftwareInfo(r *http.Request) ([]*report.SoftwareInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionSoftwareInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
