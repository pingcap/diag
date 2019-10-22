package server

import (
	"net/http"
	"net/http/pprof"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/handler/auth"
	"github.com/pingcap/tidb-foresight/handler/config"
	"github.com/pingcap/tidb-foresight/handler/emphasis"
	"github.com/pingcap/tidb-foresight/handler/inspection"
	"github.com/pingcap/tidb-foresight/handler/instance"
	"github.com/pingcap/tidb-foresight/handler/logs"
	"github.com/pingcap/tidb-foresight/handler/profile"
	"github.com/pingcap/tidb-foresight/handler/report"
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

	fn.Plugin(auth.Auth)

	// auth
	//r.HandleFunc("/api/v1/login", s.login).Methods("POST")
	r.Handle("/api/v1/login", auth.Login(s.config)).Methods("POST")
	r.Handle("/api/v1/me", auth.Me(s.config)).Methods("GET")
	r.HandleFunc("/api/v1/logout", auth.Logout).Methods("GET", "POST", "DELETE")

	// instance
	r.Handle("/api/v1/instances", instance.ListInstance(s.model)).Methods("GET")
	r.Handle("/api/v1/instances", instance.CreateInstance(s.config, s.model)).Methods("POST")
	r.Handle("/api/v1/instances/{id}", instance.GetInstance(s.model)).Methods("GET")
	r.Handle("/api/v1/instances/{id}", instance.DeleteInstance(s.config, s.model, s.scheduler)).Methods("DELETE")
	r.Handle("/api/v1/instances/{id}/components", instance.ListComponent(s.model)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/config", config.GetConfig(s.model)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/config", config.SetConfig(s.model, s.scheduler)).Methods("PUT")
	r.Handle("/api/v1/instances/{id}/inspections", inspection.ListInspection(s.model)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/inspections", inspection.CreateInspection(s.config, s.model, s.worker)).Methods("POST")
	r.Handle("/api/v1/instances/{id}/perfprofiles", profile.ListProfile(s.config, s.model)).Methods("GET")
	r.Handle("/api/v1/instances/{id}/perfprofiles", profile.CreateProfile(s.config, s.model, s.worker)).Methods("POST")

	// logs
	r.Handle("/api/v1/loginstances", logs.ListInstance(s.config, s.model)).Methods("GET")
	r.Handle("/api/v1/logfiles", logs.ListFile(s.config, s.model)).Methods("GET")
	r.Handle("/api/v1/loginstances/{id}/logs", logs.SearchLog(s.config, s.searcher)).Methods("GET")
	r.Handle("/api/v1/loginstances/{id}", logs.UploadLog(s.config)).Methods("PUT")
	r.Handle("/api/v1/logfiles/{id}/logs", logs.SearchLog(s.config, s.searcher)).Methods("GET")
	r.Handle("/api/v1/logfiles", logs.ImportLog(s.config, s.model, s.worker)).Methods("POST")
	r.Handle("/api/v1/loginstances/{id}.tar.gz", logs.ExportLog(s.config)).Methods("GET")

	// inspection
	r.Handle("/api/v1/inspections", inspection.ListAllInspection(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}.tar.gz", inspection.ExportInspection(s.config)).Methods("GET")
	r.Handle("/api/v1/inspections", inspection.ImportInspection(s.config, s.model, s.worker)).Methods("POST")
	r.Handle("/api/v1/inspections/{id}", inspection.GetInspection(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}", inspection.UploadInspection(s.config)).Methods("PUT")
	r.Handle("/api/v1/inspections/{id}", inspection.DeleteInspection(s.config, s.model)).Methods("DELETE")
	r.Handle("/api/v1/inspections/{id}", inspection.UpdateInspectionEscapedLeft(s.model)).Methods("PUT")

	// emphasis
	api := r.PathPrefix("/api/v1").Subrouter()

	{
		// List
		api.Handle("/emphasis", emphasis.Unimplemented(s.config)).Methods("GET")
		// Upload and import local reports.
		api.Handle("/emphasis", emphasis.Unimplemented(s.config)).Methods("POST")
		// List with instance_id
		api.Handle("/instances/{InstanceId}/emphasis", emphasis.Unimplemented(s.config)).Methods("GET")
		// Download resource from
		api.Handle("/emphasis/{uuid}.tar.gz", emphasis.Unimplemented(s.config)).Methods("GET")
		// Generate Report
		api.Handle("/instances/{InstanceId}/emphasis", emphasis.Unimplemented(s.config)).Methods("POST")
		// Upload emphasis
		api.Handle("/api/v1/emphasis/{id}", emphasis.Unimplemented(s.config)).Methods("PUT")
		// Get Emphasis by id
		api.Handle("/api/v1/emphasis/{id}", emphasis.Unimplemented(s.config)).Methods("GET")
		// Delete emphasis by id
		api.Handle("/api/v1/emphasis/{id}", emphasis.Unimplemented(s.config)).Methods("DELETE")
	}

	// report
	r.Handle("/api/v1/inspections/{id}/symptom", report.Symptom(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/basic", report.BasicInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/dbinfo", report.DBInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/slowlog", report.SlowLog(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/alert", report.AlertInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/config", report.ConfigInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/dmesg", report.Dmesg(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/hardware", report.HardwareInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/network", report.NetworkInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/resource", report.ResourceInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/topology", report.TopologyInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/software", report.SoftwareInfo(s.model)).Methods("GET")
	r.Handle("/api/v1/inspections/{id}/ntp", report.NtpInfo(s.model)).Methods("GET")

	// profiles
	r.Handle("/api/v1/perfprofiles", profile.ListAllProfile(s.config, s.model)).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}.tar.gz", inspection.ExportInspection(s.config)).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}", profile.GetProfile(s.config, s.model)).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}/{component}/{address}/{type}/{file}", profile.GetProfileItem(s.config)).Methods("GET")
	r.Handle("/api/v1/perfprofiles/{id}", inspection.UploadInspection(s.config)).Methods("PUT")
	r.Handle("/api/v1/perfprofiles", inspection.ImportInspection(s.config, s.model, s.worker)).Methods("POST")
	r.Handle("/api/v1/perfprofiles/{id}", inspection.DeleteInspection(s.config, s.model)).Methods("DELETE")

	// metric
	r.PathPrefix("/api/v1/metric/").HandlerFunc(s.metric)

	// all others starts with /api/v1/ are 404
	r.PathPrefix("/api/v1/").HandlerFunc(http.NotFound)

	// web
	r.PathPrefix("/").Handler(s.web("/", path.Join(s.config.Home, "web")))

	return httpRequestMiddleware(r)
}
