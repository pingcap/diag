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

type createInstanceHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

func CreateInstance(c *bootstrap.ForesightConfig, m model.Model) http.Handler {
	return &createInstanceHandler{c, m}
}

const MAX_FILE_SIZE = 32 * 1024 * 1024

func (h *createInstanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	byType := r.URL.Query().Get("by")
	//var fnPtr func(req* requestInstance, r *http.Request) (*model.Instance, utils.StatusError)
	switch byType {
	case "file":
		fn.Wrap(h.createInstanceByFile).ServeHTTP(w, r)
	case "text":
		fn.Wrap(h.createInstanceByJson).ServeHTTP(w, r)
	default:
		log.Error("Bad Request for 'by' in createInstance(r *http.Request)")
	}

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

func (h *createInstanceHandler) createInstanceByJson(req *requestInstance, r *http.Request) (*model.Instance, utils.StatusError) {
	uid := uuid.New().String()
	req.Status = "pending"

	instance := &model.Instance{Uuid: uid, User: h.c.User.Name, CreateTime: time.Now(), Status: "pending"}
	err := h.m.CreateInstance(instance)
	if err != nil {
		log.Error("create instance: ", err)
		return nil, utils.DatabaseInsertError
	}

	go func() {
		instance2 := parseTopologyByRequest(req)
		if instance2.Status == "success" {
			data, err := json.Marshal(instance)
			if err != nil {
				log.Error(err)
				return
			}
			err = utils.SaveFile(bytes.NewReader(data), path.Join(h.c.Home, "topology", instance.Uuid+".json"))
			if err != nil {
				log.Error("save topology file: ", err)
				return
			}
		}
		instance2.User = h.c.User.Name
		instance2.Name = req.ClusterName
		instance2.CreateTime = instance.CreateTime
		instance2.Uuid = uid
		if err := h.m.UpdateInstance(instance2); err != nil {
			log.Error("update instance:", err)
			return
		}
	}()

	return instance, nil
}

func (h *createInstanceHandler) createInstanceByFile(r *http.Request) (*model.Instance, utils.StatusError) {
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

func parseTopologyByRequest(result *requestInstance) *model.Instance {
	var tidb, tikv, pd, prometheus, grafana []string
	instance := &model.Instance{Status: "success"}
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

func parseTopology(topo []byte) *model.Instance {
	var result requestInstance

	err := json.Unmarshal(topo, &result)
	if err != nil {
		log.Error("exception on parse topology: ", err)
		instance := &model.Instance{Status: "success"}
		instance.Status = "exception"
		instance.Message = "集群拓扑解析异常"
		return instance
	}

	return parseTopologyByRequest(&result)
}
