package config

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Scheduler interface {
	Reload() error
}

type setConfigHandler struct {
	m model.Model
	s Scheduler
}

func SetConfig(m model.Model, s Scheduler) http.Handler {
	return &setConfigHandler{m, s}
}

func (h *setConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.setInstanceConfig).ServeHTTP(w, r)
}

func (h *setConfigHandler) setInstanceConfig(c *model.Config, r *http.Request) (*utils.SimpleResponse, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	if instanceId != c.InstanceId {
		return nil, utils.ParamsMismatch
	}

	if err := h.m.SetInstanceConfig(c); err != nil {
		log.Error("set instance config: ", err)
		return nil, utils.DatabaseUpdateError
	}

	if err := h.s.Reload(); err != nil {
		log.Error("reload auto sched task:", err)
		return nil, utils.SystemOpError
	}

	return nil, nil
}
