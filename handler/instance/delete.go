package instance

import (
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Scheduler interface {
	Reload() error
}

type deleteInstanceHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
	s Scheduler
}

func DeleteInstance(c *bootstrap.ForesightConfig, m model.Model, s Scheduler) http.Handler {
	return &deleteInstanceHandler{c, m, s}
}

func (h *deleteInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.deleteInstance).ServeHTTP(w, r)
}

func (h *deleteInstanceHandler) deleteInstance(r *http.Request) (*model.Instance, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	var e utils.StatusError
	
	if err := os.Remove(path.Join(h.c.Home, "inventory", uuid+".ini")); err != nil {
		// ini may not exists when creating by json, so here it's valid if the file not exists.
		if !os.IsNotExist(err) {
			e = utils.DatabaseDeleteError
		}
	}

	if err := os.Remove(path.Join(h.c.Home, "topology", uuid+".json")); err != nil {
		log.Error("delete inventory failed: ", err)
		// because topology.json may not exists since unsuccessful initial
		// so do nothing here
	}

	if err := h.m.DeleteInstanceConfig(uuid); err != nil {
		log.Error("delete instance config failed:", err)
		return nil, utils.DatabaseDeleteError
	}

	if err := h.s.Reload(); err != nil {
		log.Error("reload auto sched task:", err)
		return nil, utils.SystemOpError
	}

	if err := h.m.DeleteInstance(uuid); err != nil {
		log.Error("delete instance failed:", err)
		e = utils.DatabaseDeleteError
	}

	if e != nil {
		return nil, e
	} else {
		return nil, nil
	}
}
