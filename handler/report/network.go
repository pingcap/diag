package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getNetworkInfoHandler struct {
	m model.Model
}

func NetworkInfo(m model.Model) http.Handler {
	return &getNetworkInfoHandler{m}
}

func (h *getNetworkInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionNetworkInfo).ServeHTTP(w, r)
}

func (h *getNetworkInfoHandler) getInspectionNetworkInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionNetworkInfo(inspectionId)
	if err != nil {
		log.Error("get inspection network info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       info,
	}, nil
}
