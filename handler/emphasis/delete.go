package emphasis

import (
	"net/http"
	"os"
	"path"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/model/emphasis"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// Get emphasis by uuid
type deleteEmphasisHandler struct {
	m model.Model
	c *bootstrap.ForesightConfig
}

func DeleteEmphasis(m model.Model, c *bootstrap.ForesightConfig) http.Handler {
	return &deleteEmphasisHandler{m, c}
}

func (h *deleteEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.deleteEmphasis).ServeHTTP(w, r)
}

func (h *deleteEmphasisHandler) deleteEmphasis(r *http.Request) (*emphasis.Emphasis, utils.StatusError) {
	inspectionId := helper.LoadRouterVar(r, "uuid")
	uuid := inspectionId
	// TODO: the code below is replicated, please find a way to unify them.
	// ignore any error on remove inspection files
	os.RemoveAll(path.Join(h.c.Home, "inspection", uuid))
	os.RemoveAll(path.Join(h.c.Home, "package", uuid+".tar.gz"))
	os.RemoveAll(path.Join(h.c.Home, "profile", uuid))
	os.RemoveAll(path.Join(h.c.Home, "remote-log", uuid))

	if err := h.m.DeleteEmphasis(inspectionId); err != nil {
		log.Error("delete inspection:", err)
		return nil, utils.DatabaseQueryError
	} else {
		return nil, nil
	}
}
