package emphasis

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	helper "github.com/pingcap/tidb-foresight/handler/utils"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

// TODO：find a place to unify all emphasis
const CreateType = "emphasis"

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
	Start time.Time `json:"investgating_start"`
	End   time.Time `json:"investgating_end"`

	Problem string `json:"investgating_problem"`
}

func (h *createEmphasisHandler) collectEmphasis(start, end time.Time, instanceId, inspectionId string, instance *model.Instance) error {

	cmd := exec.Command(
		h.c.Collector,
		fmt.Sprintf("--home=%s", h.c.Home),
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--begin=%s", start.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
		// TODO: update it and make it more than db info.
		"--items=dbinfo",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+h.c.User.Name,
		"INSPECTION_TYPE=emphasis",
		"CLUSTER_CREATE_TIME="+instance.CreateTime.Format(time.RFC3339),
	)
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", h.c.Collector, ": ", err)
		return err
	}
	return nil
}

func (h *createEmphasisHandler) createEmphasis(req *createEmphasisRequest, r *http.Request) (*model.Emphasis, utils.StatusError) {
	instanceId := helper.LoadRouterVar(r, "instance_id")
	newUuid := uuid.New().String()

	emp := &model.Emphasis{
		Uuid:        newUuid,
		InstanceId:  instanceId,
		CreatedTime: time.Now(),

		InvestgatingStart:   req.Start,
		InvestgatingEnd:     req.End,
		InvestgatingProblem: req.Problem,
		Status:              "running",
	}

	// instance, err
	instance, err := h.m.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseQueryError)
	}

	insp := emp.CorrespondInspection()
	inspectionId := insp.Uuid

	err = h.m.SetInspection(insp)

	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, helper.GormErrorMapper(err, utils.DatabaseInsertError)
	}

	go func() {
		if err := h.collectEmphasis(req.Start, req.End, instanceId, inspectionId, instance); err != nil {
			log.Error("profile ", inspectionId, ": ", err)
			insp.Status = "exception"
			insp.Message = "profile failed"
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
