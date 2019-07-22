package server

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os/exec"
	"path"
)

func (s *Server) profileSingleProcess(instanceId, inspectionId, ip, port string) error {
	cmd := exec.Command(
		s.config.Collector,
		fmt.Sprintf("--instance-id=%s", inspectionId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--inventory=%s", path.Join(s.config.Home, "inventory", instanceId+".ini")),
		fmt.Sprintf("--topology=%s", path.Join(s.config.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--dest=%s", path.Join(s.config.Home, "inspection", inspectionId)),
		fmt.Sprintf("--collect=profile:%s:%s", ip, port),
	)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", s.config.Collector, ": ", err)
		return err
	}
	return nil
}

func (s *Server) profileAllProcess(instanceId, inspectionId string) error {
	cmd := exec.Command(
		s.config.Collector,
		fmt.Sprintf("--instance-id=%s", inspectionId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--inventory=%s", path.Join(s.config.Home, "inventory", instanceId+".ini")),
		fmt.Sprintf("--topology=%s", path.Join(s.config.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--dest=%s", path.Join(s.config.Home, "inspection", inspectionId)),
		"--collect=profile",
	)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", s.config.Collector, ": ", err)
		return err
	}
	return nil
}

func (s *Server) createProfile(r *http.Request) (*model.Inspection, error) {
	ip := r.URL.Query().Get("ip")
	port := r.URL.Query().Get("port")
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()

	inspection := &model.Inspection{
		Uuid:       inspectionId,
		InstanceId: instanceId,
		Status:     "running",
		Type:       "manual",
	}
	err := s.model.SetInspection(inspection)
	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go func() {
		var err error
		if ip != "" && port != "" {
			err = s.profileSingleProcess(instanceId, inspectionId, ip, port)
		} else {
			err = s.profileAllProcess(instanceId, inspectionId)
		}
		if err != nil {
			log.Error("profile ", inspectionId, ": ", err)
			return
		}
		err = s.analyze(inspectionId)
		if err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			return
		}
	}()

	return inspection, nil
}

func (s *Server) listProfiles(r *http.Request) ([]*model.Profile, error) {
	instanceId := mux.Vars(r)["id"]

	profiles, err := s.model.ListProfiles(instanceId)
	if err != nil {
		log.Error("list profiles: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}

	return profiles, nil
}
