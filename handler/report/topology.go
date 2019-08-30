package report

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getTopologyInfoHandler struct {
	m model.Model
}

func TopologyInfo(m model.Model) http.Handler {
	return &getTopologyInfoHandler{m}
}

func (h *getTopologyInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionTopologyInfo).ServeHTTP(w, r)
}

func (h *getTopologyInfoHandler) getInspectionTopologyInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionTopologyInfo(inspectionId)
	if err != nil {
		log.Error("get inspection Topology info:", err)
		return nil, utils.DatabaseQueryError
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       info,
	}, nil
}
