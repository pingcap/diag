package inspection

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type updateHandler struct {
	m model.Model
}

func UpdateInspectionEscapedLeft(m model.Model) http.Handler {
	return &updateHandler{m}
}

func (h *updateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.updateInspectionEscapedLeft).ServeHTTP(w, r)
}

func (h *updateHandler) updateInspectionEscapedLeft(r *http.Request) (*model.Inspection, utils.StatusError) {
	uuid := mux.Vars(r)["id"]
	leftSecStr := mux.Vars(r)["left"]
	leftSec, err := strconv.ParseInt(leftSecStr, 10, 32)
	if err != nil {
		log.Error(fmt.Sprintf("argument left should be an int32, but got %s", leftSecStr), err)
		return nil, utils.ParamsMismatch
	}
	// `ParseInt` parse with bitSize 32, so it's safe to pass `int32(leftSec)`.
	if err := h.m.UpdateInspectionEstimateLeftSec(uuid, int32(leftSec)); err != nil {
		log.Error("update inspection estimate left seconds: ", err)
		return nil, utils.DatabaseUpdateError
	}

	return nil, nil
}
