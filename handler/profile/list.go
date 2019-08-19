package profile

import (
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listProfileHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func ListProfile(c *bootstrap.ForesightConfig, m model.Model) http.Handler {
	return &listProfileHandler{c, m}
}

func (h *listProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listProfile).ServeHTTP(w, r)
}

func (h *listProfileHandler) listProfile(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	profiles, total, err := h.m.ListProfiles(instanceId, page, size, path.Join(h.c.Home, "profile"))
	if err != nil {
		log.Error("list profiles: ", err)
		return nil, utils.DatabaseQueryError
	}

	return utils.NewPaginationResponse(total, profiles), nil
}
