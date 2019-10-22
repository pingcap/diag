package emphasis

import (
	"net/http"

	"github.com/pingcap/fn"
	hepler "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// List All
type listAllEmphasisHandler struct {
	m model.Model
}

type listAllEmphasisByInstanceHandler struct {
	m model.Model
}

func ListAllEmphasis(m model.Model) http.Handler {
	return &listAllEmphasisHandler{m}
}

func ListAllEmphasisByInstance(m model.Model) http.Handler {
	return &listAllEmphasisByInstanceHandler{m}
}

func (h *listAllEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listAllEmphasis).ServeHTTP(w, r)
}

func (h *listAllEmphasisByInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listAllEmphasisByInstance).ServeHTTP(w, r)
}

func (h *listAllEmphasisByInstanceHandler) listAllEmphasisByInstance(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	page, size := hepler.LoadHttpPaging(r)
	instanceId := hepler.LoadRouterVar(r, "instance_id")

	emphasis, total, err := h.m.ListAllEmphasisOfInstance(page, size, instanceId)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.DatabaseQueryError
	}
	return utils.NewPaginationResponse(total, emphasis), nil
}

func (h *listAllEmphasisHandler) listAllEmphasis(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	page, size := hepler.LoadHttpPaging(r)
	emphasis, total, err := h.m.ListAllEmphasis(page, size)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.DatabaseQueryError
	}
	return utils.NewPaginationResponse(total, emphasis), nil
}
