package report

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getSoftwareInfoHandler struct {
	m model.Model
}

func SoftwareInfo(m model.Model) http.Handler {
	return &getSoftwareInfoHandler{m}
}

func (h *getSoftwareInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionSoftwareInfo).ServeHTTP(w, r)
}

func (h *getSoftwareInfoHandler) getInspectionSoftwareInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionSoftwareInfo(inspectionId)
	if err != nil {
		log.Error("get inspection software version:", err)
		return nil, utils.DatabaseQueryError
	}

	conclusions := make([]map[string]interface{}, 0)
	data := make([]map[string]interface{}, 0)
	for idx, comp := range info {
		// use the first one as compare base
		if idx == 0 {
			data = append(data, map[string]interface{}{
				"node_ip":   comp.NodeIp,
				"component": comp.Component,
				"version":   comp.Version,
			})
			continue
		}

		// strip prefix "v"
		pv := info[idx-1].Version
		if strings.HasPrefix(pv, "v") {
			pv = pv[1:]
		}
		cv := comp.Version
		if strings.HasPrefix(cv, "v") {
			cv = cv[1:]
		}

		if pv == cv {
			data = append(data, map[string]interface{}{
				"node_ip":   comp.NodeIp,
				"component": comp.Component,
				"version":   comp.Version,
			})
		} else {
			conclusions = append(conclusions, map[string]interface{}{
				"status":  "abnormal",
				"message": fmt.Sprintf("version of component %s on %s not the same with previous", comp.Component, comp.NodeIp),
			})
			data = append(data, map[string]interface{}{
				"node_ip":   comp.NodeIp,
				"component": comp.Component,
				"version": map[string]interface{}{
					"value":    comp.Version,
					"abnormal": true,
					"message":  "not identity with previous",
				},
			})
		}
	}

	return map[string]interface{}{
		"conclusion": conclusions,
		"data":       data,
	}, nil
}
