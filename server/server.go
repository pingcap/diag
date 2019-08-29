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
	"github.com/pingcap/tidb-foresight/server/scheduler"
	"github.com/pingcap/tidb-foresight/server/worker"
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config    *bootstrap.ForesightConfig
	model     model.Model
	router    http.Handler
	worker    worker.Worker
	searcher  search.Searcher
	scheduler scheduler.Scheduler
}

type ErrorMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func New(config *bootstrap.ForesightConfig, db db.DB) *Server {
	model := model.New(db)
	worker := worker.New(config, model)
	s := &Server{
		config:    config,
		model:     model,
		worker:    worker,
		searcher:  search.NewSearcher(),
		scheduler: scheduler.New(model, worker),
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

	s.router = s.CreateRouter()

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
			fmt.Sprintf("--threshold=%d", s.config.Log.Threshold),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Error("syncer:", err)
		}
	}()

	log.Info("start listen on ", address)

	s.scheduler.Reload()
	return http.ListenAndServe(address, s.router)
}
