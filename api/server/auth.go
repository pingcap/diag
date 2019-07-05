package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func (s *Server) authFunc(next func(http.ResponseWriter, *http.Request)) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("enter auth")
		next(w, r)
	})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("enter auth")
		next.ServeHTTP(w, r)
	})
}
