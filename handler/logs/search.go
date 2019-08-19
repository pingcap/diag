package logs

import (
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/log/parser"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type searchLogHandler struct {
	c *bootstrap.ForesightConfig
	s LogSearcher
}

func SearchLog(c *bootstrap.ForesightConfig, s LogSearcher) http.Handler {
	return &searchLogHandler{c, s}
}

func (h *searchLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.searchLog).ServeHTTP(w, r)
}

func (h *searchLogHandler) searchLog(r *http.Request) (*LogResult, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	search := r.URL.Query().Get("search")
	token := r.URL.Query().Get("token")
	level := r.URL.Query().Get("level")
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 10
	}

	begin := time.Time{} // long long ago.
	end := time.Now()
	if bt, e := time.Parse(time.RFC3339, r.URL.Query().Get("start_time")); e == nil {
		begin = bt
	}
	if et, e := time.Parse(time.RFC3339, r.URL.Query().Get("end_time")); e == nil {
		end = et
	}

	iter, token, err := h.s.Search(
		path.Join(h.c.Home, "remote-log", instanceId),
		begin, end, level, search, token,
	)
	if err != nil {
		log.Error("open log: ", err)
		return nil, utils.FileOpError
	}
	if iter == nil {
		return nil, utils.TargetObjectNotFound
	}

	logs := []*LogItem{}
	for i := 0; i < int(limit); i++ {
		if l, err := iter.Next(); err != nil {
			log.Error("search log: ", err)
			return nil, utils.FileOpError
		} else if l == nil {
			// no more logs
			log.Info("search to end")
			token = ""
			if err := iter.Close(); err != nil {
				log.Error("close log:", err)
			}
			break
		} else {
			logs = append(logs, logFromSearch(l.Get()))
		}
	}

	return &LogResult{
		Token: token,
		Logs:  logs,
	}, nil
}

func logFromSearch(l *parser.LogItem) *LogItem {
	item := &LogItem{
		Host:      l.Host,
		Port:      l.Port,
		Component: l.Component,
		File:      l.File,
		Time:      l.Time,
		Content:   string(l.Line),
	}

	switch l.Level {
	case -1:
		item.Level = "SLOWLOG"
	case parser.LevelFATAL:
		item.Level = "FATAL"
	case parser.LevelERROR:
		item.Level = "ERROR"
	case parser.LevelWARN:
		item.Level = "WARN"
	case parser.LevelINFO:
		item.Level = "INFO"
	case parser.LevelDEBUG:
		item.Level = "DEBUG"
	}

	return item
}
