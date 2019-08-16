package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getDmesgHandler struct {
	m DmesgGeter
}

func Dmesg(m DmesgGeter) http.Handler {
	return &getDmesgHandler{m}
}

func (h *getDmesgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionDmesg).ServeHTTP(w, r)
}

func (h *getDmesgHandler) getInspectionDmesg(r *http.Request) ([]*report.DmesgLog, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionDmesg(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
