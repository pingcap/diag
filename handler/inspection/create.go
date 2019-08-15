package inspection

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type createInspectionHandler struct {
	c *bootstrap.ForesightConfig
	m InspectionCreator
}

func CreateInspection(c *bootstrap.ForesightConfig, m InspectionCreator) http.Handler {
	return &createInspectionHandler{c, m}
}

func (h *createInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createInspection).ServeHTTP(w, r)
}

func (h *createInspectionHandler) createInspection(r *http.Request, c *model.Config) (*model.Inspection, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()

	inspection := &model.Inspection{
		Uuid:       inspectionId,
		InstanceId: instanceId,
		Status:     "running",
		Type:       "manual",
	}
	err := h.m.SetInspection(inspection)
	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go func() {
		if err := h.collectInspection(instanceId, inspectionId, c); err != nil {
			log.Error("collect ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "collect failed"
			h.m.SetInspection(inspection)
			return
		}

		if err := utils.Analyze(h.c.Analyzer, h.c.Home, h.c.Influx.Endpoint, h.c.Prometheus.Endpoint, inspectionId); err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "analyze failed"
			h.m.SetInspection(inspection)
			return
		}
	}()

	return inspection, nil
}

func (h *createInspectionHandler) collectInspection(instanceId, inspectionId string, config *model.Config) error {
	instance, err := h.m.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return err
	}

	from := time.Now().Add(time.Duration(-10) * time.Minute)
	to := time.Now()
	if len(config.ManualSchedRange) > 0 {
		from = config.ManualSchedRange[0]
	}
	if len(config.ManualSchedRange) > 1 {
		to = config.ManualSchedRange[1]
	}

	items := []string{"metric", "basic", "dbinfo", "config", "profile"}
	if config != nil {
		if config.CollectHardwareInfo {
			//	items = append(items, "hardware")
		}
		if config.CollectSoftwareInfo {
			//	items = append(items, "software")
		}
		if config.CollectLog {
			//	items = append(items, "log")
		}
		if config.CollectDemsg {
			//	items = append(items, "demsg")
		}
	}

	cmd := exec.Command(
		h.c.Collector,
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(h.c.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(h.c.Home, "inspection")),
		fmt.Sprintf("--collect=%s", strings.Join(items, ",")),
		fmt.Sprintf("--log-spliter=%s", h.c.Spliter),
		// TODO: use time range in config
		fmt.Sprintf("--begin=%s", from.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", to.Format(time.RFC3339)),
	)
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+h.c.User.Name,
		"CLUSTER_CREATE_TIME="+instance.CreateTime.Format(time.RFC3339),
		"INSPECTION_TYPE=manual",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)
	err = cmd.Run()
	if err != nil {
		log.Error("run ", h.c.Collector, ": ", err)
		return err
	}
	return nil
}
