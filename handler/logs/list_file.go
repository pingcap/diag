package logs

import (
	"io/ioutil"
	"net/http"
	"path"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listFileHandler struct {
	c *bootstrap.ForesightConfig
	m LogFileLister
}

func ListFile(c *bootstrap.ForesightConfig, m LogFileLister) http.Handler {
	return &listFileHandler{c, m}
}

func (h *listFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listLogFile).ServeHTTP(w, r)
}

func (h *listFileHandler) listLogFile(r *http.Request) ([]*model.LogEntity, utils.StatusError) {
	ls, err := ioutil.ReadDir(path.Join(h.c.Home, "remote-log"))
	if err != nil {
		log.Error("read dir: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read dir")
	}
	logs := []string{}
	for _, l := range ls {
		logs = append(logs, l.Name())
	}

	entities, err := h.m.ListLogFiles(logs)
	if err != nil {
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_DB_ERROR", "error on query database")
	}
	return entities, nil
}
