package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getNetworkInfoHandler struct {
	m NetworkInfoGeter
}

func NetworkInfo(m NetworkInfoGeter) http.Handler {
	return &getNetworkInfoHandler{m}
}

func (h *getNetworkInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionNetworkInfo).ServeHTTP(w, r)
}

func (h *getNetworkInfoHandler) getInspectionNetworkInfo(r *http.Request) ([]*report.NetworkInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionNetworkInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
