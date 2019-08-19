package inspection

import (
	"net/http"
	"strconv"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listAllInspectionHandler struct {
	m model.Model
}

func ListAllInspection(m model.Model) http.Handler {
	return &listAllInspectionHandler{m}
}

func (h *listAllInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listAllInspection).ServeHTTP(w, r)
}

func (h *listAllInspectionHandler) listAllInspection(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}
	inspections, total, err := h.m.ListAllInspections(page, size)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}
	return utils.NewPaginationResponse(total, inspections), nil
}
