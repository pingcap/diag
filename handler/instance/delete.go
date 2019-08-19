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

type InstanceDeletor interface {
	DeleteInstance(uuid string) error
}

type deleteInstanceHandler struct {
	c *bootstrap.ForesightConfig
	m InstanceDeletor
}

func DeleteInstance(c *bootstrap.ForesightConfig, m InstanceDeletor) http.Handler {
	return &deleteInstanceHandler{c, m}
}

func (h *deleteInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.deleteInstance).ServeHTTP(w, r)
}

func (h *deleteInstanceHandler) deleteInstance(r *http.Request) (*model.Instance, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	var e utils.StatusError
	if err := os.Remove(path.Join(h.c.Home, "inventory", uuid+".ini")); err != nil {
		log.Error("delete inventory failed: ", err)
		e = utils.DatabaseDeleteError
	}

	if err := os.Remove(path.Join(h.c.Home, "topology", uuid+".json")); err != nil {
		log.Error("delete inventory failed: ", err)
		// because topology.json may not exists since unsuccessful initial
		// so do nothing here
	}

	if err := h.m.DeleteInstance(uuid); err != nil {
		log.Error("delete instance failed: ", err)
		e = utils.DatabaseDeleteError
	}

	if e != nil {
		return nil, e
	} else {
		return nil, nil
	}
}
