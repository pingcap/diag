package instance

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getInstanceHandler struct {
	m model.Model
}

func GetInstance(m model.Model) http.Handler {
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
		return nil, utils.DatabaseQueryError
	}

	return instance, nil
}
