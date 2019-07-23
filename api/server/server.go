package server

import (
	"context"
	"database/sql"
	"net/http"
	"path"
	"time"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/searcher"
	"github.com/pingcap/tidb-foresight/syncer"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config   *bootstrap.ForesightConfig
	model    *model.Model
	Router   http.Handler
	searcher *searcher.Searcher
}

type ErrorMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewServer(config *bootstrap.ForesightConfig, db *sql.DB) *Server {
	s := &Server{
		config:   config,
		model:    model.NewModel(db),
		searcher: searcher.NewSearcher(),
	}

	fn.SetErrorEncoder(func(ctx context.Context, err error) interface{} {
		if e, ok := err.(utils.StatusError); ok {
			return &ErrorMessage{
				Status:  e.Status(),
				Message: e.Error(),
			}
		} else {
			return &ErrorMessage{
				Status:  "UNKNOWN_ERROR",
				Message: err.Error(),
			}
		}
	})

	s.Router = s.CreateRouter()

	return s
}

func (s *Server) Run() error {
	log.Info("start listen on ", s.config.Address)

	go syncer.Sync(path.Join(s.config.Home, "topology"), path.Join(s.config.Home, "remote-log"), 5*time.Minute, 10)

	return http.ListenAndServe(s.config.Address, s.Router)
}
