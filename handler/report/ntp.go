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

const NTP_TRESHHOLD = 500.0

type getNtpInfoHandler struct {
	m model.Model
}

func NtpInfo(m model.Model) http.Handler {
	return &getNtpInfoHandler{m}
}

func (h *getNtpInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionNtpInfo).ServeHTTP(w, r)
}

func (h *getNtpInfoHandler) getInspectionNtpInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	ntps, err := h.m.GetInspectionNtpInfo(inspectionId)
	if err != nil {
		log.Error("get inspection slow log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	conclusions := make([]map[string]interface{}, 0)
	data := make([]map[string]interface{}, 0)
	for _, ntp := range ntps {
		if ntp.Offset > NTP_TRESHHOLD {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  "abnormal",
				"message": fmt.Sprintf("ntp offset of node %s exceeded the threshold (500ms)", ntp.NodeIp),
			})
			data = append(data, map[string]interface{}{
				"node_ip": ntp.NodeIp,
				"value": map[string]interface{}{
					"value":    ntp.Offset,
					"abnormal": true,
					"message":  "exceeded the threshold (500ms)",
				},
			})
		} else {
			data = append(data, map[string]interface{}{
				"node_ip": ntp.NodeIp,
				"value":   ntp.Offset,
			})
		}
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       data,
	}, nil
}
