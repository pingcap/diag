package profile

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type getProfileHandler struct {
	c *bootstrap.ForesightConfig
	m ProfileGeter
}

func GetProfile(c *bootstrap.ForesightConfig, m ProfileGeter) http.Handler {
	return &getProfileHandler{c, m}
}

func (h *getProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getProfile).ServeHTTP(w, r)
}

func (h *getProfileHandler) getProfile(r *http.Request) (*model.Profile, utils.StatusError) {
	profileId := mux.Vars(r)["id"]

	if profile, err := h.m.GetProfile(profileId, path.Join(h.c.Home, "profile")); err != nil {
		log.Error("get profile detail:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	} else {
		return profile, nil
	}
}
