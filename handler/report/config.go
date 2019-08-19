package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getConfigInfoHandler struct {
	m model.Model
}

func ConfigInfo(m model.Model) http.Handler {
	return &getConfigInfoHandler{m}
}

func (h *getConfigInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionConfigInfo).ServeHTTP(w, r)
}

func (h *getConfigInfoHandler) getInspectionConfigInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionConfigInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       info,
	}, nil
}
