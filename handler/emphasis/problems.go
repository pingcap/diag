package emphasis

import (
	"github.com/pingcap/tidb-foresight/wrapper/db"
	"net/http"

	"github.com/pingcap/fn"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/model/emphasis"
	"github.com/pingcap/tidb-foresight/utils"
)

type loadProblemsHandler struct {
	m model.Model
}

func LoadAllProblems(m model.Model) http.Handler {
	return &loadProblemsHandler{m}
}

func (h *loadProblemsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.listAllProblems).ServeHTTP(w, r)
}

func (h *loadProblemsHandler) listAllProblems(r *http.Request) (map[string]interface{}, utils.StatusError) {
	uuid := helper.LoadRouterVar(r, "uuid")
	emp, err := h.m.GetEmphasis(uuid)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, utils.TargetObjectNotFound
		}
		return nil, utils.DatabaseQueryError
	}
	problems, err := h.m.LoadAllProblems(emp)
	if err != nil {
		return nil, utils.DatabaseQueryError
	}
	return emphasis.ArrayToSymptoms(problems), nil
}
