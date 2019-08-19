package inspection

import (
	"net/http"
	"os"
	"path"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type deleteInspectionHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func DeleteInspection(c *bootstrap.ForesightConfig, m model.Model) http.Handler {
	return &deleteInspectionHandler{c, m}
}

func (h *deleteInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.deleteInspection).ServeHTTP(w, r)
}

func (h *deleteInspectionHandler) deleteInspection(r *http.Request) (*model.Inspection, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	// ignore any error on remove inspection files
	os.RemoveAll(path.Join(h.c.Home, "inspection", uuid))
	os.RemoveAll(path.Join(h.c.Home, "package", uuid+".tar.gz"))
	os.RemoveAll(path.Join(h.c.Home, "profile", uuid))
	os.RemoveAll(path.Join(h.c.Home, "remote-log", uuid))

	if err := h.m.DeleteInspection(uuid); err != nil {
		log.Error("delete inspection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "error on delete data")
	}

	return nil, nil
}
