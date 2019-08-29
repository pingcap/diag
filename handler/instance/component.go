package instance

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listComponentHandler struct {
	m model.Model
}

func ListComponent(m model.Model) http.Handler {
	return &listComponentHandler{m}
}

func (h *listComponentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listComponent).ServeHTTP(w, r)
}

func (h *listComponentHandler) listComponent(r *http.Request) ([]*model.Component, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	instance, err := h.m.GetInstance(uuid)
	if err != nil {
		log.Error("query instance: ", err)
		return nil, utils.DatabaseQueryError
	}

	comps := []*model.Component{}

	comps = append(comps, components(instance.Pd, "pd")...)
	comps = append(comps, components(instance.Tidb, "tidb")...)
	comps = append(comps, components(instance.Tikv, "tikv")...)

	return comps, nil
}

func components(str, name string) []*model.Component {
	comps := []*model.Component{}
	for _, comp := range strings.Split(str, ",") {
		ss := strings.Split(comp, ":")
		if len(ss) != 2 {
			log.Errorf("%s stored with wrong format", name)
			continue
		}
		comps = append(comps, &model.Component{
			Name: name,
			Ip:   ss[0],
			Port: ss[1],
		})
	}
	return comps
}
