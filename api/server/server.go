package server

import (
	"database/sql"
	"net/http"

	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config *bootstrap.ForesightConfig
	model  *model.Model
	Router http.Handler
}

func NewServer(config *bootstrap.ForesightConfig, db *sql.DB) *Server {
	s := &Server{
		config: config,
		model:  model.NewModel(db),
	}

	s.Router = s.CreateRouter()

	return s
}

func (s *Server) Run() error {
	log.Info("start listen on ", s.config.Address)

	return http.ListenAndServe(s.config.Address, s.Router)
}
