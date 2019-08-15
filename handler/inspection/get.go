package inspection

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getInspectionHandler struct {
	m InspectionGeter
}

func GetInspection(m InspectionGeter) http.Handler {
	return &getInspectionHandler{m}
}

func (h *getInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspection).ServeHTTP(w, r)
}

func (h *getInspectionHandler) getInspection(r *http.Request) (*model.Inspection, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]

	if inspection, err := h.m.GetInspection(inspectionId); err != nil {
		log.Error("get inspection:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	} else {
		return inspection, nil
	}
}
