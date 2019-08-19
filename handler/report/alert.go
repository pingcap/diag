package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getAlertInfoHandler struct {
	m model.Model
}

func AlertInfo(m model.Model) http.Handler {
	return &getAlertInfoHandler{m}
}

func (h *getAlertInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionAlertInfo).ServeHTTP(w, r)
}

func (h *getAlertInfoHandler) getInspectionAlertInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionAlertInfo(inspectionId)
	if err != nil {
		log.Error("get inspection alert info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       info,
	}, nil
}
