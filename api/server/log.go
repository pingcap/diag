package server

import (
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/searcher"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
)

type LogResult struct {
	Token string          `json:"token"`
	Logs  []*searcher.Item `json:"logs"`
}

func (s *Server) listLogs() ([]string, error) {
	ls, err := ioutil.ReadDir(path.Join(s.config.Home, "remote-log"))
	if err != nil {
		log.Error("read dir: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read dir")
	}
	logs := []string{}
	for _, l := range ls {
		logs = append(logs, l.Name())
	}
	return logs, nil
}

func (s *Server) searchLog(r *http.Request) (*LogResult, error) {
	instanceId := mux.Vars(r)["id"]
	search := r.URL.Query().Get("search")
	token := r.URL.Query().Get("token")
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 10
	}

	iter, token, err := s.searcher.Search(path.Join(s.config.Home, "remote-log", instanceId), search, token)
	if err != nil {
		log.Error("open log: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on open file")
	}
	if iter == nil {
		return nil, utils.NewForesightError(http.StatusNotFound, "NOT_FOUND", "token not found")
	}

	logs := []*searcher.Item{}
	for i := 0; i < int(limit); i++ {
		if l, err := iter.Next(); err != nil {
			log.Error("search log: ", err)
			return nil, err
		} else if l == nil {
			// no more logs
			log.Info("search to end")
			token = ""
			if err := iter.Close(); err != nil {
				log.Error("close log:", err)
			}
			break
		} else {
			logs = append(logs, l)
		}
	}

	return &LogResult{
		Token: token,
		Logs:  logs,
	}, nil
}
