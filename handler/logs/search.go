package logs

import (
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/log/item"
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
		if err == utils.TargetObjectNotFound {
			return nil, utils.TargetObjectNotFound
		} else {
			log.Error("open log: ", err)
			return nil, utils.FileOpError
		}
	}

	logs := []*LogItem{}
	for i := 0; i < int(limit); i++ {
		if l, err := iter.Next(); err != nil {
			if err == io.EOF {
				// no more logs
				log.Info("search to end")
				token = ""
				if err := iter.Close(); err != nil {
					log.Error("close log:", err)
				}
				break
			} else {
				log.Error("search log: ", err)
				return nil, utils.FileOpError
			}
		} else {
			logs = append(logs, logFromSearch(l))
		}
	}

	return &LogResult{
		Token: token,
		Logs:  logs,
	}, nil
}

func logFromSearch(l item.Item) *LogItem {
	it := &LogItem{
		Host:      l.GetHost(),
		Port:      l.GetPort(),
		Component: l.GetComponent(),
		File:      l.GetFileName(),
		Time:      l.GetTime(),
		Content:   string(l.GetContent()),
	}

	switch l.GetLevel() {
	case -1:
		it.Level = "SLOWLOG"
	case item.LevelFATAL:
		it.Level = "FATAL"
	case item.LevelERROR:
		it.Level = "ERROR"
	case item.LevelWARN:
		it.Level = "WARN"
	case item.LevelINFO:
		it.Level = "INFO"
	case item.LevelDEBUG:
		it.Level = "DEBUG"
	}

	return it
}
