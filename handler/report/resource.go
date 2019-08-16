package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getResourceInfoHandler struct {
	m model.Model
}

func ResourceInfo(m model.Model) http.Handler {
	return &getResourceInfoHandler{m}
}

func (h *getResourceInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionResourceInfo).ServeHTTP(w, r)
}

func (h *getResourceInfoHandler) getInspectionResourceInfo(r *http.Request) ([]*model.ResourceInfo, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionResourceInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
