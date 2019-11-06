package emphasis

import (
	"github.com/pingcap/tidb-foresight/wrapper/db"
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
	log.Info("getEmphasis(r *http.Request) load with uuid ", inspectionId)
	if inspection, err := h.m.GetEmphasis(inspectionId); err != nil {
		log.Error("get inspection:", err)
		if db.IsNotFound(err) {
			return nil, utils.TargetObjectNotFound
		}
		return nil, utils.DatabaseQueryError
	} else {
		problems, err := h.m.LoadAllProblems(inspection)
		if err != nil {
			return nil, utils.DatabaseQueryError
		}
		inspection.RelatedProblems = problems
		return inspection, nil
	}
}
