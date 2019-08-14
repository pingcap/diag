package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model/report"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

func (s *Server) getInspectionBasicInfo(r *http.Request) (*report.BasicInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionBasicInfo(inspectionId)
	if err != nil {
		log.Error("get inspection basic info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionAlertInfo(r *http.Request) ([]*report.AlertInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionAlertInfo(inspectionId)
	if err != nil {
		log.Error("get inspection alert info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionConfigInfo(r *http.Request) ([]*report.ConfigInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionConfigInfo(inspectionId)
	if err != nil {
		log.Error("get inspection config info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionDBInfo(r *http.Request) ([]*report.DBInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionDBInfo(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionDmesg(r *http.Request) ([]*report.DmesgLog, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionDmesg(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionHardwareInfo(r *http.Request) ([]*report.HardwareInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionHardwareInfo(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionSymptom(r *http.Request) ([]*report.Symptom, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionSymtoms(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionDmesgInfo(r *http.Request) ([]*report.DmesgLog, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionDmesg(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionNetworkInfo(r *http.Request) ([]*report.NetworkInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionNetworkInfo(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionResourceInfo(r *http.Request) ([]*report.ResourceInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionResourceInfo(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}

func (s *Server) getInspectionSlowLogInfo(r *http.Request) ([]*report.SlowLogInfo, error) {
	inspectionId := mux.Vars(r)["id"]
	info, err := s.model.GetInspectionSlowLog(inspectionId)
	if err != nil {
		log.Error("get inspection db info:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}

	return info, nil
}
