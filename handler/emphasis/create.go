package emphasis

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"

	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"

	"net/http"
)

type Worker interface {
	Collect() error
	Analyze() error
}

//type DiagnoseWorker interface {
//	Collect(inspectionId, inspectionType string, config *model.Config) error
//	Analyze(inspectionId string) error
//}

// 1. 启动具体的 collector 和 analyzer
// 2. 利用 gorm 创建 model
type createEmphasisHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
	w Worker
}

func CreateEmphasis(c *bootstrap.ForesightConfig, m model.Model, w Worker) http.Handler {
	return &createEmphasisHandler{c, m, w}
}

func (h *createEmphasisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createEmphasis).ServeHTTP(w, r)
}

type createEmphasisRequest struct {
	Start time.Time `json:"investgating_start"`
	End   time.Time `json:"investgating_end"`

	Problem string `json:"investgating_problem"`
}

func (h *createEmphasisHandler) createEmphasis(r *http.Request, c *model.Config) (*model.Emphasis, utils.StatusError) {
	instanceId := helper.LoadRouterVar(r, "instance_id")
	newUuid := uuid.New().String()
	c.InstanceId = instanceId

	var req createEmphasisRequest

	if err := helper.LoadJsonFromHttpBody(r, &req); err != nil {
		return nil, utils.ParamsMismatch
	}
	emp := &model.Emphasis{
		Uuid:        newUuid,
		InstanceId:  instanceId,
		CreatedTime: time.Now(),

		InvestgatingStart:   req.Start,
		InvestgatingEnd:     req.End,
		InvestgatingProblem: req.Problem,
	}

	if err := h.m.CreateEmphasis(emp); err != nil {
		log.Error("create emphasis error ", err)
		return nil, utils.DatabaseDeleteError
	}

	// TODO: implement the part of collect and analyze.
	go func() {
		if err := h.w.Collect(); err != nil {
			panic("implement me")
			return
		}

		if err := h.w.Analyze(); err != nil {
			panic("implement me")
			return
		}
	}()

	return emp, nil
}
