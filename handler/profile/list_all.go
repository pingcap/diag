package profile

import (
	"net/http"
	"path"
	"strconv"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listAllProfileHandler struct {
	c *bootstrap.ForesightConfig
	m AllProfileLister
}

func ListAllProfile(c *bootstrap.ForesightConfig, m AllProfileLister) http.Handler {
	return &listAllProfileHandler{c, m}
}

func (h *listAllProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listAllProfile).ServeHTTP(w, r)
}

func (h *listAllProfileHandler) listAllProfile(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	profiles, total, err := h.m.ListAllProfiles(page, size, path.Join(h.c.Home, "profile"))
	if err != nil {
		log.Error("list profiles: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}
	return utils.NewPaginationResponse(total, profiles), nil
}
