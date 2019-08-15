package logs

import (
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type exportLogHandler struct {
	c *bootstrap.ForesightConfig
}

func ExportLog(c *bootstrap.ForesightConfig) http.Handler {
	return &exportLogHandler{c}
}

func (h *exportLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.exportLog(w, r)
}

func (h *exportLogHandler) exportLog(w http.ResponseWriter, r *http.Request) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	begin := time.Now().Add(time.Duration(-1) * time.Hour)
	end := time.Now()

	if bt, e := time.Parse(time.RFC3339, r.URL.Query().Get("begin")); e == nil {
		begin = bt
	}
	if et, e := time.Parse(time.RFC3339, r.URL.Query().Get("end")); e == nil {
		end = et
	}

	if err := utils.CollectLog(
		h.c.Collector, h.c.Home, h.c.User.Name, instanceId, inspectionId, begin, end,
	); err != nil {
		log.Error("collect log:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := utils.PackInspection(h.c.Home, inspectionId); err != nil {
		log.Error("pack: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	localFile, err := os.Open(path.Join(h.c.Home, "package", inspectionId+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+inspectionId+".tar.gz")
	io.Copy(w, localFile)
}
