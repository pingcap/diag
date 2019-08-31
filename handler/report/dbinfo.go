package report

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getDBInfoHandler struct {
	m model.Model
}

func DBInfo(m model.Model) http.Handler {
	return &getDBInfoHandler{m}
}

func (h *getDBInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionDBInfo).ServeHTTP(w, r)
}

func (h *getDBInfoHandler) getInspectionDBInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionDBInfo(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.DatabaseQueryError
	}

	count := 0
	for _, tb := range info {
		if tb.Index.GetTag("status") != "" {
			count++
		}
	}
	conclusions := make([]map[string]interface{}, 0)
	if count != 0 {
		conclusions = append(conclusions, map[string]interface{}{
			"status":  "error",
			"message": fmt.Sprintf("there are %d tables are abnormal", count),
		})
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       info,
	}, nil
}
