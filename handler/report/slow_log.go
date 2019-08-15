package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getSlowLogHandler struct {
	m SlowLogGeter
}

func SlowLog(m SlowLogGeter) http.Handler {
	return &getSlowLogHandler{m}
}

func (h *getSlowLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionSlowLog).ServeHTTP(w, r)
}

func (h *getSlowLogHandler) getInspectionSlowLog(r *http.Request) ([]*report.SlowLogInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionSlowLog(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
