package server

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (s *Server) listInstance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")

	instances, err := s.model.ListInstance()
	if err != nil {
		log.Error("Query instance list failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "DB_ERROR", "message": "查询数据库时发生错误"}`))
		return
	}

	jsonContent, err := json.Marshal(instances)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "MARSHAL_JSON_ERROR", "message": "序列化json时发生错误"}`))
		log.Error("序列化json时发生错误", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonContent)
}
