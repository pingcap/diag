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
	data := make([]map[string]interface{}, 0)
	for _, res := range info {
		var max, avg interface{}

		if res.Max.GetTag("status") == "abnormal" {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  "abnormal",
				"message": fmt.Sprintf("%s/max resource utilization/%s too high", res.Name, res.Duration),
			})
			max = map[string]interface{}{
				"value":    res.Max.GetValue(),
				"abnormal": true,
				"message":  "too high",
			}
		} else {
			max = res.Max.GetValue()
		}

		if res.Avg.GetTag("status") == "abnormal" {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  "abnormal",
				"message": fmt.Sprintf("%s/avg resource utilization/%s too high", res.Name, res.Duration),
			})
			avg = map[string]interface{}{
				"value":    res.Avg.GetValue(),
				"abnormal": true,
				"message":  "too high",
			}
		} else {
			avg = res.Avg.GetValue()
		}

		data = append(data, map[string]interface{}{
			"name":     res.Name,
			"duration": res.Duration,
			"max":      max,
			"avg":      avg,
		})
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       data,
	}, nil
}
