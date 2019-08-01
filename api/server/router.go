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

	// auth
	r.HandleFunc("/api/v1/login", s.login).Methods("POST")
	r.Handle("/api/v1/me", fn.Wrap(s.me)).Methods("GET")
	r.HandleFunc("/api/v1/logout", s.logout).Methods("GET", "POST", "DELETE")

	// instance
	r.Handle("/api/v1/instances", fn.Wrap(s.listInstance)).Methods("GET")
	r.Handle("/api/v1/instances", fn.Wrap(s.createInstance)).Methods("POST")
	r.Handle("/api/v1/instances/{id}", fn.Wrap(s.getInstance)).Methods("GET")
	r.Handle("/api/v1/instances/{id}", fn.Wrap(s.deleteInstance)).Methods("DELETE")
	r.Handle("/api/v1/instances/{id}/config", fn.Wrap(s.getInstanceConfig)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/config", fn.Wrap(s.updateInstanceConfig)).Methods("PUT")
	r.Handle("/api/v1/instances/{id}/inspections", fn.Wrap(s.listInspections)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/inspections", fn.Wrap(s.createInspection)).Methods("POST")
	r.Handle("/api/v1/instances/{id}/perfprofiles", fn.Wrap(s.listProfiles)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/perfprofiles", fn.Wrap(s.createProfile)).Methods("POST")

	// logs
	r.Handle("/api/v1/loginstances", fn.Wrap(s.listLogInstances)).Methods("GET")
	r.Handle("/api/v1/logfiles", fn.Wrap(s.listLogFiles)).Methods("GET")
	r.Handle("/api/v1/loginstances/{id}/logs", fn.Wrap(s.searchLog)).Methods("GET")
	r.Handle("/api/v1/loginstances/{id}", fn.Wrap(s.uploadLog)).Methods("PUT")
	r.Handle("/api/v1/logfiles/{id}/logs", fn.Wrap(s.searchLog)).Methods("GET")
	// upload log inspection (dba)
	r.Handle("/api/v1/logfiles", fn.Wrap(s.importLog)).Methods("POST")
	// download log inspection (user)
	r.HandleFunc("/api/v1/loginstances/{id}.tar.gz", s.exportLog).Methods("GET")

	// metric
	r.PathPrefix("/api/v1/metric/").HandlerFunc(s.metric)

	// inspection
	r.Handle("/api/v1/inspections", fn.Wrap(s.listAllInspections)).Methods("GET")
	r.HandleFunc("/api/v1/inspections/{id}.tar.gz", s.exportInspection).Methods("GET")
	r.Handle("/api/v1/inspections", fn.Wrap(s.importInspection)).Methods("POST")
	r.Handle("/api/v1/inspections/{id}", fn.Wrap(s.getInspectionDetail)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}", fn.Wrap(s.uploadInspection)).Methods("PUT")
	r.Handle("/api/v1/inspections/{id}", fn.Wrap(s.deleteInspection)).Methods("DELETE")

	// profiles
	r.Handle("/api/v1/perfprofiles", fn.Wrap(s.listAllProfiles)).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}", fn.Wrap(s.getProfileDetail)).Methods("GET")
	r.HandleFunc("/api/v1/perfprofiles/{id}/{component}/{address}/{type}/{file}", s.getProfile).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}", fn.Wrap(s.uploadInspection)).Methods("PUT")
	r.HandleFunc("/api/v1/perfprofiles/{id}.tar.gz", s.exportInspection).Methods("GET")
	r.Handle("/api/v1/perfprofiles", fn.Wrap(s.importInspection)).Methods("POST")
	r.Handle("/api/v1/perfprofiles/{id}", fn.Wrap(s.deleteInspection)).Methods("DELETE")

	// other
	r.Handle("/ping", fn.Wrap(s.ping)).Methods("GET")
	r.Handle("/static/", s.static("/static/", path.Join(s.config.Home, "static")))

	return httpRequestMiddleware(r)
}
