package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/log/search"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config   *bootstrap.ForesightConfig
	model    model.Model
	Router   http.Handler
	searcher search.Searcher
}

type ErrorMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewServer(config *bootstrap.ForesightConfig, db db.DB) *Server {
	s := &Server{
		config:   config,
		model:    model.New(db),
		searcher: search.NewSearcher(),
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
				Message: "make sure your request is valid",
			}
		}
	})

	s.Router = s.CreateRouter()

	return s
}

func (s *Server) Run(address string) error {
	// sync log from cluster
	go func() {
		log.Info("start sync log from cluster")
		cmd := exec.Command(
			s.config.Syncer,
			fmt.Sprintf("--topo=%s", path.Join(s.config.Home, "topology")),
			fmt.Sprintf("--target=%s", path.Join(s.config.Home, "remote-log")),
			fmt.Sprintf("--interval=%d", s.config.Log.Interval),
			fmt.Sprintf("--bwlimit=%d", s.config.Log.Bwlimit),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Error("syncer:", err)
		}
	}()

	log.Info("start listen on ", address)

	return http.ListenAndServe(address, s.Router)
}
