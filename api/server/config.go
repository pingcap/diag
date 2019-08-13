package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/api/model"
	"github.com/pingcap/tidb-foresight/api/utils"
	log "github.com/sirupsen/logrus"
)

func (s *Server) getInstanceConfig(r *http.Request) (*model.Config, error) {
	instanceId := mux.Vars(r)["id"]

	config, err := s.model.GetInstanceConfig(instanceId)
	if err == nil {
		if config == nil {
			return nil, utils.NewForesightError(http.StatusNotFound, "NOT_FOUND", "target instance config not found")
		} else {
			return config, nil
		}
	} else {
		log.Error("get instance config: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query db")
	}
}

func (s *Server) updateInstanceConfig(c *model.Config, r *http.Request) (*utils.SimpleResponse, error) {
	instanceId := mux.Vars(r)["id"]
	if instanceId != c.InstanceId {
		return nil, utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "instance id mismatch")
	}

	err := s.model.SetInstanceConfig(c)
	if err != nil {
		log.Error("update instance config: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_UPDATE_ERROR", "error on update database")
	}

	return nil, nil
}
