package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getDmesgHandler struct {
	m model.Model
}

func Dmesg(m model.Model) http.Handler {
	return &getDmesgHandler{m}
}

func (h *getDmesgHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionDmesg).ServeHTTP(w, r)
}

func (h *getDmesgHandler) getInspectionDmesg(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionDmesg(inspectionId)
	if err != nil {
		log.Error("get inspection dmesg:", err)
		return nil, utils.DatabaseQueryError
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       info,
	}, nil
}
