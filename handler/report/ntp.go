package report

import (
	"fmt"
	"math"
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
		log.Error("get inspection ntp info:", err)
		return nil, utils.DatabaseQueryError
	}

	conclusions := make([]map[string]interface{}, 0)
	data := make([]map[string]interface{}, 0)
	for _, ntp := range ntps {
		if math.Abs(ntp.Offset) > NTP_TRESHHOLD {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  "abnormal",
				"message": fmt.Sprintf("ntp offset of node %s exceeded the threshold (500ms)", ntp.NodeIp),
			})
			data = append(data, map[string]interface{}{
				"node_ip": ntp.NodeIp,
				"offset": map[string]interface{}{
					"value":    ntp.Offset,
					"abnormal": true,
					"message":  "exceeded the threshold (500ms)",
				},
			})
		} else {
			data = append(data, map[string]interface{}{
				"node_ip": ntp.NodeIp,
				"offset":  ntp.Offset,
			})
		}
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       data,
	}, nil
}
