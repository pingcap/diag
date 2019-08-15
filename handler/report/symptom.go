package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getSymptomHandler struct {
	m SymptomGeter
}

func Symptom(m SymptomGeter) http.Handler {
	return &getSymptomHandler{m}
}

func (h *getSymptomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionSymptom).ServeHTTP(w, r)
}

func (h *getSymptomHandler) getInspectionSymptom(r *http.Request) ([]*report.Symptom, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionSymptoms(inspectionId)
	if err != nil {
		log.Error("get inspection symptoms:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
