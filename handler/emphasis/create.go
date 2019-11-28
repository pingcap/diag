package emphasis

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/utils/debug_printer"
	log "github.com/sirupsen/logrus"
)

type Worker interface {
	Collect(inspectionId string, config *model.Config, env map[string]string) error
	Analyze(inspectionId string) error
}

// 1. Start collector and analyzer
// 2. Using gorm to create model
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
	Start   time.Time `json:"investgating_start"`
	End     time.Time `json:"investgating_end"`
	Problem string    `json:"investgating_problem"`

	//TODO: ask for frontend if this is possible.
	//Config *model.Config `json:"config"`
}

func (h *createEmphasisHandler) createEmphasis(req *createEmphasisRequest, r *http.Request) (*model.Emphasis, utils.StatusError) {
	log.Infof("(h *createEmphasisHandler) createEmphasis received request %v", debug_printer.FormatJson(req))

	instanceId := helper.LoadRouterVar(r, "instance_id")
	newUuid := uuid.New().String()

	instance, err := h.m.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseQueryError)
	}

	emp := &model.Emphasis{
		Uuid:                newUuid,
		InstanceId:          instanceId,
		CreatedTime:         time.Now(),
		InvestgatingStart:   req.Start,
		InvestgatingEnd:     req.End,
		InvestgatingProblem: req.Problem,
		Status:              "running",
		InstanceName:        instance.Name,
		User:                instance.User,
	}

	insp := emp.CorrespondInspection()
	inspectionId := insp.Uuid
	log.Infof("(h *createEmphasisHandler) createEmphasis create insp %v", debug_printer.FormatJson(insp))
	if err := h.m.SetInspection(insp); err != nil {
		log.Error("set inspection: ", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseInsertError)
	}

	config, err := h.m.GetInstanceConfig(instanceId)
	if err != nil {
		log.Error("get instance config:", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseQueryError)
	}

	go func() {
		if err := h.w.Collect(inspectionId, config, map[string]string{
			"INSPECTION_TYPE": "emphasis",
			"PROBLEM":         req.Problem,
		}); err != nil {
			log.Error("collect ", inspectionId, ": ", err)
			insp.Status = "exception"
			insp.Message = "collect failed"
			h.m.SetInspection(insp)
			return
		}
		if err := h.w.Analyze(inspectionId); err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			insp.Status = "exception"
			insp.Message = "analyze failed"
			h.m.SetInspection(insp)
			return
		}
	}()

	return emp, nil
}
