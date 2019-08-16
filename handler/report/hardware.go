package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getHardwareInfoHandler struct {
	m model.Model
}

func HardwareInfo(m model.Model) http.Handler {
	return &getHardwareInfoHandler{m}
}

func (h *getHardwareInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionHardwareInfo).ServeHTTP(w, r)
}

func (h *getHardwareInfoHandler) getInspectionHardwareInfo(r *http.Request) ([]*model.HardwareInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionHardwareInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
