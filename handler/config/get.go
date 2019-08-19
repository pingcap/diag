package config

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getConfigHandler struct {
	m model.Model
}

func GetConfig(m model.Model) http.Handler {
	return &getConfigHandler{m}
}

func (h *getConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInstanceConfig).ServeHTTP(w, r)
}

func (h *getConfigHandler) getInstanceConfig(r *http.Request) (*model.Config, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]

	if config, err := h.m.GetInstanceConfig(instanceId); err == nil {
		if config == nil {
			return nil, utils.NewForesightError(http.StatusNotFound, "NOT_FOUND", "target instance config not found")
		} else {
			return config, nil
		}
	} else {
		log.Error("get instance config: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query db")
	}
}
