package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getConfigInfoHandler struct {
	m ConfigInfoGeter
}

func ConfigInfo(m ConfigInfoGeter) http.Handler {
	return &getConfigInfoHandler{m}
}

func (h *getConfigInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionConfigInfo).ServeHTTP(w, r)
}

func (h *getConfigInfoHandler) getInspectionConfigInfo(r *http.Request) ([]*report.ConfigInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionConfigInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
