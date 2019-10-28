package emphasis

import (
	"net/http"

	"github.com/pingcap/fn"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/model/emphasis"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// Get emphasis by uuid
type getEmphasisHandler struct {
	m model.Model
}

func GetEmphasis(m model.Model) http.Handler {
	return &getEmphasisHandler{m}
}

func (h *getEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.getEmphasis).ServeHTTP(w, r)
}

func (h *getEmphasisHandler) getEmphasis(r *http.Request) (*emphasis.Emphasis, utils.StatusError) {
	inspectionId := helper.LoadRouterVar(r, "uuid")

	if inspection, err := h.m.GetEmphasis(inspectionId); err != nil {
		log.Error("get inspection:", err)
		return nil, utils.DatabaseQueryError
	} else {
		return inspection, nil
	}
}
