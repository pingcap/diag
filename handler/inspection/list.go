package inspection

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type listInspectionHandler struct {
	m InspectionLister
}

func ListInspection(m InspectionLister) http.Handler {
	return &listInspectionHandler{m}
}

func (h *listInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listInspection).ServeHTTP(w, r)
}

func (h *listInspectionHandler) listInspection(r *http.Request) (*utils.PaginationResponse, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	inspections, total, err := h.m.ListInspections(instanceId, page, size)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}

	return utils.NewPaginationResponse(total, inspections), nil
}
