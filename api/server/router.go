package server

import (
	"net/http"
	"net/http/pprof"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	log "github.com/sirupsen/logrus"
)

type traceResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *traceResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func httpRequestMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Request : %s - %s - %s", r.RemoteAddr, r.Method, r.URL)
		start := time.Now()
		tw := &traceResponseWriter{w, http.StatusOK}
		h.ServeHTTP(tw, r)
		log.Infof("Response [%d] : %s - %s - %s (%.3f sec)",
			tw.statusCode, r.RemoteAddr, r.Method, r.URL, time.Since(start).Seconds())
	})
}

// AttachProfiler attachs pprofs
func AttachProfiler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	// Manually add support for paths linked to by index page at /debug/pprof/
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
}

// CreateRouter creates router for api
func (s *Server) CreateRouter() http.Handler {
	r := mux.NewRouter()
	AttachProfiler(r)

	fn.Plugin(s.auth)

	// instance
	r.Handle("/instances", fn.Wrap(s.listInstance)).Methods("GET")
	r.Handle("/instances", fn.Wrap(s.createInstance)).Methods("POST")
	r.Handle("/instances/{id}", fn.Wrap(s.deleteInstance)).Methods("DELETE")

	// auth
	r.HandleFunc("/login", s.login).Methods("GET")
	r.Handle("/me", fn.Wrap(s.me)).Methods("GET")
	r.HandleFunc("/logout", s.logout).Methods("GET")
	r.HandleFunc("/ping", s.ping).Methods("GET")
	r.HandleFunc("/upload", s.upload)
	r.HandleFunc("/config", s.inspectionConfig)
	r.Handle("/static/", s.static("/static/", path.Join(s.config.Home, "static")))

	return httpRequestMiddleware(r)
}
