package server

import (
	"net/http"
)

func (s *Server) inspection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/json")
	w.WriteHeader(http.StatusOK)
}
