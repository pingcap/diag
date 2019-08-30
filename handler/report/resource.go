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

func (h *getResourceInfoHandler) getInspectionResourceInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionResourceInfo(inspectionId)
	if err != nil {
		log.Error("get inspection resource info:", err)
		return nil, utils.DatabaseQueryError
	}

	conclusions := make([]map[string]interface{}, 0)
	for _, res := range info {
		if res.Max.GetTag("status") != "" {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  res.Max.GetTag("status"),
				"message": res.Max.GetTag("message"),
			})
		}

		if res.Avg.GetTag("status") != "" {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  res.Avg.GetTag("status"),
				"message": res.Avg.GetTag("message"),
			})
		}
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       info,
	}, nil
}
