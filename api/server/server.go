package server

import (
	"database/sql"
	"net/http"
	"path"

	"github.com/pingcap/tidb-foresight/bootstrap"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	home   string
	config *bootstrap.ForesightConfig
	db     *sql.DB
}

func NewServer(config *bootstrap.ForesightConfig, db *sql.DB) *Server {
	s := &Server{
		config: config,
		db:     db,
	}

	http.Handle("/ping", s.authFunc(s.ping))
	http.HandleFunc("/upload", s.upload)
	http.Handle("/static/", s.static("/static/", path.Join(config.Home, "static")))

	return s
}

func (s *Server) Run() error {
	log.Info("start listen on ", s.config.Address)

	return http.ListenAndServe(s.config.Address, nil)
}
