package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

func (s *Server) profileAllProcess(instanceId, inspectionId string) error {
	cmd := exec.Command(
		s.config.Collector,
		fmt.Sprintf("--instance-id=%s", inspectionId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(s.config.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(s.config.Home, "inspection")),
		"--collect=profile",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		cmd.Env,
		"FORESIGHT_USER="+s.config.User.Name,
		"INSPECTION_TYPE=profile",
	)
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", s.config.Collector, ": ", err)
		return err
	}
	return nil
}

func (s *Server) createProfile(r *http.Request) (*model.Profile, error) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()

	instance, err := s.model.GetInstance(instanceId)
	if err != nil {
		log.Error("get instance:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}
	inspection := &model.Inspection{
		Uuid:         inspectionId,
		InstanceId:   instanceId,
		InstanceName: instance.Name,
		Status:       "running",
		Type:         "profile",
	}

	err = s.model.SetInspection(inspection)
	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go func() {
		err := s.profileAllProcess(instanceId, inspectionId)
		if err != nil {
			log.Error("profile ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "profile failed"
			s.model.SetInspection(inspection)
			return
		}
		err = s.analyze(inspectionId)
		if err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			inspection.Status = "exception"
			inspection.Message = "analyze failed"
			s.model.SetInspection(inspection)
			return
		}
	}()

	return &model.Profile{
		Uuid:         inspectionId,
		InstanceName: instance.Name,
		Status:       "running",
	}, nil
}

func (s *Server) listProfiles(r *http.Request) (*utils.PaginationResponse, error) {
	instanceId := mux.Vars(r)["id"]
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	profiles, total, err := s.model.ListProfiles(instanceId, page, size, path.Join(s.config.Home, "profile"))
	if err != nil {
		log.Error("list profiles: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}

	return utils.NewPaginationResponse(total, profiles), nil
}

func (s *Server) listAllProfiles(r *http.Request) (*utils.PaginationResponse, error) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	profiles, total, err := s.model.ListAllProfiles(page, size, path.Join(s.config.Home, "profile"))
	if err != nil {
		log.Error("list profiles: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}
	return utils.NewPaginationResponse(total, profiles), nil
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["id"]
	comp := mux.Vars(r)["component"]
	addr := mux.Vars(r)["address"]
	tp := mux.Vars(r)["type"]
	file := mux.Vars(r)["file"]
	fpath := path.Join(s.config.Home, "profile", uuid, comp+"-"+addr, tp, file)

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Info("profile not found:", fpath)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 NOT FOUND"))
		return
	}

	localFile, err := os.Open(fpath)
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	io.Copy(w, localFile)
}

func (s *Server) getProfileDetail(r *http.Request) (*model.Profile, error) {
	profileId := mux.Vars(r)["id"]

	if profile, err := s.model.GetProfileDetail(profileId, path.Join(s.config.Home, "profile")); err != nil {
		log.Error("get profile detail:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	} else {
		return profile, nil
	}
}
