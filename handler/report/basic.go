package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getBasicInfoHandler struct {
	m model.Model
}

func BasicInfo(m model.Model) http.Handler {
	return &getBasicInfoHandler{m}
}

func (h *getBasicInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionBasicInfo).ServeHTTP(w, r)
}

func (h *getBasicInfoHandler) getInspectionBasicInfo(r *http.Request) (*model.BasicInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionBasicInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
