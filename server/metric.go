package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (s *Server) metric(w http.ResponseWriter, r *http.Request) {
	u := s.config.Prometheus.Endpoint + "/api/v1"
	url, err := url.Parse(u)
	if err != nil {
		log.Panic("parse url:", err)
	}

	r.URL.Host = url.Host
	r.URL.Path = strings.Replace(r.URL.Path, "/api/v1/metric", "", 1)
	r.Host = url.Host

	proxy := httputil.NewSingleHostReverseProxy(url)

	proxy.ServeHTTP(w, r)
}
