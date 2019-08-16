package instance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type InstanceCreator interface {
	CreateInstance(instance *model.Instance) error
	UpdateInstance(instance *model.Instance) error
}

type createInstanceHandler struct {
	c *bootstrap.ForesightConfig
	m InstanceCreator
}

func CreateInstance(c *bootstrap.ForesightConfig, m InstanceCreator) http.Handler {
	return &createInstanceHandler{c, m}
}

func (h *createInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createInstance).ServeHTTP(w, r)
}

func (h *createInstanceHandler) createInstance(r *http.Request) (*model.Instance, utils.StatusError) {
	uid := uuid.New().String()

	const MAX_FILE_SIZE = 32 * 1024 * 1024
	r.ParseMultipartForm(MAX_FILE_SIZE)
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Error("retrieving file: ", err)
		return nil, utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "error on retrieving file")
	}
	defer file.Close()

	inventoryPath := path.Join(h.c.Home, "inventory", uid+".ini")
	err = utils.SaveFile(file, inventoryPath)
	if err != nil {
		log.Error("save file: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on save file")
	}

	instance := &model.Instance{Uuid: uid, User: h.c.User.Name, CreateTime: time.Now(), Status: "pending"}
	err = h.m.CreateInstance(instance)
	if err != nil {
		log.Error("create instance: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go h.importInstance(h.c.Pioneer, inventoryPath, uid)

	return instance, nil
}

func (h *createInstanceHandler) importInstance(pioneerPath, inventoryPath, instanceId string) error {
	cmd := exec.Command(pioneerPath, inventoryPath)
	log.Info(cmd.Args)

	output, err := cmd.Output()
	if err != nil {
		log.Error("error run pioneer: ", err)
		return err
	}

	instance := parseTopology(output)
	if instance.Status == "success" {
		err = utils.SaveFile(bytes.NewReader(output), path.Join(h.c.Home, "topology", instanceId+".json"))
		if err != nil {
			log.Error("save topology file: ", err)
			return err
		}
	}

	instance.Uuid = instanceId
	return h.m.UpdateInstance(instance)
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
