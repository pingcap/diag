package emphasis

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// TODO: replace this worker to the only real worker.
type Worker interface {
	Collect(inspectionId, inspectionType string, config *model.Config) error
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
}

func (h *createEmphasisHandler) createEmphasis(req *createEmphasisRequest, r *http.Request) (*model.Emphasis, utils.StatusError) {
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

	config, err := h.m.GetInstanceConfig(instanceId)
	if err != nil {
		log.Error("get instance config:", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseQueryError)
	}

	// dmesg = true to mark it.
	collectDmesgPtr := helper.LoadSelectableRouterVar(r, "dmesg")
	if collectDmesgPtr != nil {
		CollectDemsg, err := strconv.ParseBool(*collectDmesgPtr)
		if err != nil {
			config.CollectDemsg = CollectDemsg
		}
	}

	insp := emp.CorrespondInspection()
	inspectionId := insp.Uuid

	if err := h.m.SetInspection(insp); err != nil {
		log.Error("set inspection: ", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseInsertError)
	}

	go func() {
		if err := h.w.Collect(inspectionId, "emphasis", config); err != nil {
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
