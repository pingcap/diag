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
	m model.Model
}

func GetInspection(m model.Model) http.Handler {
	return &getInspectionHandler{m}
}

func (h *getInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspection).ServeHTTP(w, r)
}

func (h *getInspectionHandler) getInspection(r *http.Request) (*model.Inspection, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]

	if inspection, err := h.m.GetInspection(inspectionId); err != nil {
		log.Error("get inspection:", err)
		return nil, utils.DatabaseQueryError
	} else {
		return inspection, nil
	}
}
