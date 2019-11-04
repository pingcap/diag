package instance

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type createInstanceHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func CreateInstance(c *bootstrap.ForesightConfig, m model.Model) http.Handler {
	return &createInstanceHandler{c, m}
}

const MAX_FILE_SIZE = 32 * 1024 * 1024

func (h *createInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.createInstance).ServeHTTP(w, r)
}

// Adding instance by ini file.
func (h *createInstanceHandler) fromIniFile(r *http.Request) (*model.Instance, utils.StatusError) {
	uid := uuid.New().String()

	r.ParseMultipartForm(MAX_FILE_SIZE)
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Error("retrieving file: ", err)
		return nil, utils.NetworkError
	}
	defer file.Close()

	inventoryPath := path.Join(h.c.Home, "inventory", uid+".ini")
	err = utils.SaveFile(file, inventoryPath)
	if err != nil {
		log.Error("save file: ", err)
		return nil, utils.FileOpError
	}

	instance := &model.Instance{Uuid: uid, User: h.c.User.Name, CreateTime: time.Now(), Status: "pending"}
	err = h.m.CreateInstance(instance)
	if err != nil {
		log.Error("create instance: ", err)
		return nil, utils.DatabaseInsertError
	}

	go h.importInstance(h.c.Pioneer, inventoryPath, uid)

	return instance, nil
}

// adding by json request
func (h *createInstanceHandler) fromJsonRequest(r *http.Request) (*model.Instance, utils.StatusError) {
	return &model.Instance{}, nil
}

type requestInstance struct {
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
}

func (h *createInstanceHandler) createInstance(req *requestInstance, r *http.Request) (*model.Instance, utils.StatusError) {
	byType := r.URL.Query().Get("by")
	// TODO: remove all debug messages after debugging.
	if req == nil {
		log.Info(" createInstance got nil")
	} else {
		if data, err := json.Marshal(req); err != nil {
			log.Info(fmt.Sprintf(" createInstance got %s", string(data)))
		} else {
			log.Info(" createInstance using json.Marshal but got nil")
		}
	}

	switch byType {
	case "file":
		return h.fromIniFile(r)
	case "text":
		return h.fromJsonRequest(r)
	default:
		log.Error("Bad Request for 'by' in createInstance(r *http.Request)")
		return nil, utils.ParamsMismatch
	}
}

func (h *createInstanceHandler) importInstance(pioneerPath, inventoryPath, instanceId string) {
	cmd := exec.Command(pioneerPath, inventoryPath)
	log.Info(cmd.Args)

	output, err := cmd.Output()
	if err != nil {
		log.Error("error run pioneer: ", err)
		return
	}

	instance := parseTopology(output)
	if instance.Status == "success" {
		err = utils.SaveFile(bytes.NewReader(output), path.Join(h.c.Home, "topology", instanceId+".json"))
		if err != nil {
			log.Error("save topology file: ", err)
			return
		}
	}

	instance.Uuid = instanceId
	if err := h.m.UpdateInstance(instance); err != nil {
		log.Error("update instance:", err)
		return
	}
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
