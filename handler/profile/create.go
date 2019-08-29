package profile

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type ProfileWorker interface {
	Analyze(inspectionId string) error
}

type createProfileHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
	w ProfileWorker
}

func CreateProfile(c *bootstrap.ForesightConfig, m model.Model, w ProfileWorker) http.Handler {
	return &createProfileHandler{c, m, w}
}

func (h *createProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createProfile).ServeHTTP(w, r)
}

func (h *createProfileHandler) createProfile(r *http.Request) (*model.Profile, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	component := r.URL.Query().Get("node")

	instance, err := h.m.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return nil, utils.DatabaseQueryError
	}
	inspection := &model.Inspection{
		Uuid:         inspectionId,
		InstanceId:   instanceId,
		InstanceName: instance.Name,
		Status:       "running",
		Type:         "profile",
	}

	err = h.m.SetInspection(inspection)
	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, utils.DatabaseInsertError
	}

	go func() {
		if err := h.collectProfile(instanceId, inspectionId, strings.Trim(component, " ")); err != nil {
			log.Error("profile ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "profile failed"
			h.m.SetInspection(inspection)
			return
		}
		if err := h.w.Analyze(inspectionId); err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "analyze failed"
			h.m.SetInspection(inspection)
			return
		}
	}()

	return &model.Profile{
		Uuid:         inspectionId,
		InstanceName: instance.Name,
		Status:       "running",
	}, nil
}

func (h *createProfileHandler) collectProfile(instanceId, inspectionId, component string) error {
	option := "profile"
	if component != "" {
		option += ":" + component
	}
	cmd := exec.Command(
		h.c.Collector,
		fmt.Sprintf("--instance-id=%s", inspectionId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(h.c.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(h.c.Home, "inspection")),
		fmt.Sprintf("--collect=%s", option),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		"FORESIGHT_USER="+h.c.User.Name,
		"INSPECTION_TYPE=profile",
	)
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", h.c.Collector, ": ", err)
		return err
	}
	return nil
}
