package server

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	AutoRun struct {
		Enable   bool   `json:"enable"`
		Duration string `json:"duration"`
	} `json:"auto_run"`
	Items struct {
		Topology  bool `json:"topology"`
		BasicInfo bool `json:"basic_info"`
		DBInfo    bool `json:"db_info"`
		SlowLog   struct {
			Enable   bool   `json:"enable"`
			Duration string `json:"duration"`
		} `json:"slow_log"`
		Alert           bool `json:"alert"`
		SoftwareVersion bool `json:"software_version"`
		SoftwareConfig  bool `json:"swftware_config"`
		Ntp             bool `json:"ntp"`
		Resource        bool `json:"resource"`
		Hardware        bool `json:"hardware"`
		Network         bool `json:"network"`
	} `json:"items"`
}

func (s *Server) inspectionConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")

	config := Config{}
	config.AutoRun.Enable = s.config.Sched.Auto
	config.AutoRun.Duration = s.config.Sched.Duration

	jsonContent, err := json.Marshal(config)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "MARSHAL_JSON_ERROR", "message": "序列化json时发生错误"}`))
		log.Error("序列化json时发生错误", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonContent)
}
