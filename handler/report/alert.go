package report

import (
	"github.com/pingcap/tidb-foresight/model/report"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getAlertInfoHandler struct {
	m model.Model
}

func AlertInfo(m model.Model) http.Handler {
	return &getAlertInfoHandler{m}
}

func (h *getAlertInfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInspectionAlertInfo).ServeHTTP(w, r)
}

type AlertInfoWrapper struct {
	report.AlertInfo
	AlertCnt uint64 `json:"alert_cnt"`
}

func HandleAlertInfos(infos []*report.AlertInfo) []*AlertInfoWrapper {
	wrapperMap := make(map[string]*AlertInfoWrapper)
	for _, info := range infos {
		if v, exists := wrapperMap[info.Name]; exists {
			v.AlertCnt++
		} else {
			wrapperMap[info.Name] = &AlertInfoWrapper{*info, 1}
		}
	}
	values := make([]*AlertInfoWrapper, 1)
	for _, v := range wrapperMap {
		if v != nil {
			values = append(values, v)
		}
	}
	return values
}

func (h *getAlertInfoHandler) getInspectionAlertInfo(r *http.Request) (map[string]interface{}, utils.StatusError) {
	inspectionId := mux.Vars(r)["id"]
	info, err := h.m.GetInspectionAlertInfo(inspectionId)

	if err != nil {
		log.Error("get inspection alert info:", err)
		return nil, utils.DatabaseQueryError
	}

	return map[string]interface{}{
		"conclusion": []interface{}{},
		"data":       HandleAlertInfos(info),
	}, nil
}
