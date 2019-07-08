package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

func (s *Server) listInstance() ([]*model.Instance, error) {
	instances, err := s.model.ListInstance()
	if err != nil {
		log.Error("Query instance list failed: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "查询数据库发生错误")
	}

	return instances, nil
}

func (s *Server) createInstance(instance *model.Instance) (*utils.SimpleResponse, error) {
	err := s.model.CreateInstance(instance)
	if err != nil {
		log.Error("Create instance failed: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "插入数据时发生错误")
	}
	return utils.NewSimpleResponse("OK", "success"), nil
}

func (s *Server) deleteInstance(r *http.Request) (*utils.SimpleResponse, error) {
	uuid := mux.Vars(r)["id"]

	if err := s.model.DeleteInstance(uuid); err != nil {
		log.Error("Delete instance failed: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "删除数据时发生错误")
	}
	return utils.NewSimpleResponse("OK", "success"), nil
}
