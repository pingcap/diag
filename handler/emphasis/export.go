package emphasis

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type exportEmphasisHandler struct {
	c *bootstrap.ForesightConfig
}

func ExportEmphasis(c *bootstrap.ForesightConfig) http.Handler {
	return &exportEmphasisHandler{c}
}

func (h *exportEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.exportEmphasis(w, r)
}

func (h *exportEmphasisHandler) exportEmphasis(w http.ResponseWriter, r *http.Request) {
	uuid := helper.LoadRouterVar(r, "uuid")

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
