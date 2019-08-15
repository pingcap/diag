package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getAlertInfoHandler struct {
	m AlertInfoGeter
}

func AlertInfo(m AlertInfoGeter) http.Handler {
	return &getAlertInfoHandler{m}
}

func (h *getAlertInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionAlertInfo).ServeHTTP(w, r)
}

func (h *getAlertInfoHandler) getInspectionAlertInfo(r *http.Request) ([]*report.AlertInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionAlertInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
