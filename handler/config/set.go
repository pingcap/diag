package config

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type setConfigHandler struct {
	m ConfigSeter
}

func SetConfig(m ConfigSeter) http.Handler {
	return &setConfigHandler{m}
}

func (h *setConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.setInstanceConfig).ServeHTTP(w, r)
}

func (h *setConfigHandler) setInstanceConfig(c *model.Config, r *http.Request) (*utils.SimpleResponse, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	if instanceId != c.InstanceId {
		return nil, utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "instance id mismatch")
	}

	err := h.m.SetInstanceConfig(c)
	if err != nil {
		log.Error("set instance config: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_UPDATE_ERROR", "error on update database")
	}

	return nil, nil
}
