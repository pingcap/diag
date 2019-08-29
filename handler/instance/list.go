package instance

import (
	"net/http"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listInstanceHandler struct {
	m model.Model
}

func ListInstance(m model.Model) http.Handler {
	return &listInstanceHandler{m}
}

func (h *listInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listInstance).ServeHTTP(w, r)
}

func (h *listInstanceHandler) listInstance() ([]*model.Instance, utils.StatusError) {
	instances, err := h.m.ListInstance()
	if err != nil {
		log.Error("Query instance list: ", err)
		return nil, utils.DatabaseQueryError
	}

	return instances, nil
}
