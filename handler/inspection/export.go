package inspection

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type exportInspectionHandler struct {
	c *bootstrap.ForesightConfig
}

func ExportInspection(c *bootstrap.ForesightConfig) http.Handler {
	return &exportInspectionHandler{c}
}

func (h *exportInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.exportInspection(w, r)
}

func (h *exportInspectionHandler) exportInspection(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["id"]

	if _, err := os.Stat(path.Join(h.c.Home, "package", uuid+".tar.gz")); os.IsNotExist(err) {
		if err := utils.PackInspection(h.c.Home, uuid); err != nil {
			log.Error("pack: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	localFile, err := os.Open(path.Join(h.c.Home, "package", uuid+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	io.Copy(w, localFile)
}
