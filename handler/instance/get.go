package instance

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type InstanceGeter interface {
	GetInstance(instanceId string) (*model.Instance, error)
}

type getInstanceHandler struct {
	m InstanceGeter
}

func GetInstance(m InstanceGeter) http.Handler {
	return &getInstanceHandler{m}
}

func (h *getInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getInstance).ServeHTTP(w, r)
}

func (h *getInstanceHandler) getInstance(r *http.Request) (*model.Instance, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	instance, err := h.m.GetInstance(uuid)
	if err != nil {
		log.Error("query instance: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query database")
	}

	return instance, nil
}
