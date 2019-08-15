package profile

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	log "github.com/sirupsen/logrus"
)

type getProfileItemHandler struct {
	c *bootstrap.ForesightConfig
}

func GetProfileItem(c *bootstrap.ForesightConfig) http.Handler {
	return &getProfileItemHandler{c}
}

func (h *getProfileItemHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getProfileItem).ServeHTTP(w, r)
}

func (h *getProfileItemHandler) getProfileItem(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["id"]
	comp := mux.Vars(r)["component"]
	addr := mux.Vars(r)["address"]
	tp := mux.Vars(r)["type"]
	file := mux.Vars(r)["file"]
	fpath := path.Join(h.c.Home, "profile", uuid, comp+"-"+addr, tp, file)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Info("profile not found:", fpath)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 NOT FOUND"))
		return
	}

	localFile, err := os.Open(fpath)
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	io.Copy(w, localFile)
}
