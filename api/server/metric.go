package server

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func (s *Server) metric(w http.ResponseWriter, r *http.Request) {
	u := "http://localhost:8888" + strings.Replace(r.URL.Path, "/metric", "", 1) + "?" + r.URL.RawQuery
	url, err := url.Parse(u)
	if err != nil {
		log.Info("parse url: ", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "BAD_REQUEST", "message": "url invalid"}`))
	}

	r.URL.Scheme = url.Scheme
	r.URL.Host = url.Host
	r.URL.Path = url.Path
	r.Host = url.Host

	proxy := httputil.NewSingleHostReverseProxy(url)

	proxy.ServeHTTP(w, r)
}
