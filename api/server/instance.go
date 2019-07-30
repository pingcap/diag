package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

func (s *Server) listInstance() ([]*model.Instance, error) {
	instances, err := s.model.ListInstance()
	if err != nil {
		log.Error("Query instance list: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query database")
	}

	return instances, nil
}

func (s *Server) createInstance(r *http.Request) (*model.Instance, error) {
	uid := uuid.New().String()

        const MAX_FILE_SIZE = 32 * 1024 * 1024
        r.ParseMultipartForm(MAX_FILE_SIZE)
        file, _, err := r.FormFile("file")
        if err != nil {
                log.Error("retrieving file: ", err)
                return nil, utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "error on retrieving file")
        }
        defer file.Close()

	inventoryPath := path.Join(s.config.Home, "inventory", uid+".ini")
	err = utils.SaveFile(file, inventoryPath)
	if err != nil {
		log.Error("save file: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on save file")
	}

	err = s.model.SetInstanceConfig(model.DefaultInstanceConfig(uid))
	if err != nil {
		log.Error("create instance config: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	instance := &model.Instance{Uuid: uid, CreateTime: time.Now(), Status: "pending"}
	err = s.model.CreateInstance(instance)
	if err != nil {
		log.Error("create instance: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go s.importInstance(s.config.Pioneer, inventoryPath, uid)

	return instance, nil
}

func (s *Server) getInstance(r *http.Request) (*model.Instance, error) {
	uuid := mux.Vars(r)["id"]

	instance, err := s.model.GetInstance(uuid)
	if err != nil {
		log.Error("query instance: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query database")
	}

	return instance, nil
}

func (s *Server) deleteInstance(r *http.Request) (*utils.SimpleResponse, error) {
	uuid := mux.Vars(r)["id"]
	var e error

	if err := os.Remove(path.Join(s.config.Home, "inventory", uuid+".ini")); err != nil {
		log.Error("delete inventory failed: ", err)
		e = utils.NewForesightError(http.StatusInternalServerError, "FS_DELETE_ERROR", "error on delete file")
	}

	if err := os.Remove(path.Join(s.config.Home, "topology", uuid+".json")); err != nil {
		log.Error("delete inventory failed: ", err)
		// because topology.json may not exists since unsuccessful initial
		// so do nothing here
	}

	if err := s.model.DeleteInstanceConfig(uuid); err != nil {
		log.Error("delete instance config failed: ", err)
		e = utils.NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "error on delete data")
	}

	if err := s.model.DeleteInstance(uuid); err != nil {
		log.Error("delete instance failed: ", err)
		e = utils.NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "error on delete data")
	}

	if e != nil {
		return nil, e
	} else {
		return nil, nil
	}
}

func (s *Server) importInstance(pioneerPath, inventoryPath, instanceId string) error {
	cmd := exec.Command(pioneerPath, inventoryPath)
	log.Info(cmd.Args)

	output, err := cmd.Output()
	if err != nil {
		log.Error("error run pioneer: ", err)
		return err
	}

	instance := parseTopology(output)
	if instance.Status == "success" {
		err = utils.SaveFile(bytes.NewReader(output), path.Join(s.config.Home, "topology", instanceId+".json"))
		if err != nil {
			log.Error("save topology file: ", err)
			return err
		}
	}

	instance.Uuid = instanceId
	return s.model.UpdateInstance(instance)
}

func parseTopology(topo []byte) *model.Instance {
	result := struct {
		Status      string `json:"status"`
		Message     string `json:"message"`
		ClusterName string `json:"cluster_name"`
		Hosts       []struct {
			Ip         string `json:"ip"`
			Status     string `json:"status"`
			Message    string `json:"message"`
			Components []struct {
				Name string `json:"name"`
				Port string `json:"port"`
			} `json:"components"`
		} `json:"hosts"`
	}{}

	instance := &model.Instance{Status: "success"}
	err := json.Unmarshal(topo, &result)
	if err != nil {
		log.Error("exception on parse topology: ", err)
		instance.Status = "exception"
		instance.Message = "集群拓扑解析异常"
		return instance
	}

	var tidb, tikv, pd, prometheus, grafana []string
	for _, h := range result.Hosts {
		if h.Status == "exception" {
			instance.Status = "exception"
			instance.Message = h.Message
		}
		for _, c := range h.Components {
			switch c.Name {
			case "tidb":
				tidb = append(tidb, h.Ip+":"+c.Port)
			case "tikv":
				tikv = append(tikv, h.Ip+":"+c.Port)
			case "pd":
				pd = append(pd, h.Ip+":"+c.Port)
			case "prometheus":
				prometheus = append(prometheus, h.Ip+":"+c.Port)
			case "grafana":
				grafana = append(grafana, h.Ip+":"+c.Port)
			}
		}
	}
	instance.Name = result.ClusterName
	instance.Tidb = strings.Join(tidb, ",")
	instance.Tikv = strings.Join(tikv, ",")
	instance.Pd = strings.Join(pd, ",")
	instance.Prometheus = strings.Join(prometheus, ",")
	instance.Grafana = strings.Join(grafana, ",")

	return instance
}
