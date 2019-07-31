package server

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/searcher"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type LogResult struct {
	Token string           `json:"token"`
	Logs  []*searcher.Item `json:"logs"`
}

func (s *Server) listLogInstances(r *http.Request) (*utils.PaginationResponse, error) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	ls, err := ioutil.ReadDir(path.Join(s.config.Home, "remote-log"))
	if err != nil {
		log.Error("read dir: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read dir")
	}
	logs := []string{}
	for _, l := range ls {
		logs = append(logs, l.Name())
	}

	entities, total, err := s.model.ListLogInstances(logs, page, size)
	if err != nil {
		return nil, err
	}
	return utils.NewPaginationResponse(total, entities), nil
}

func (s *Server) listLogFiles(r *http.Request) (*utils.PaginationResponse, error) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	ls, err := ioutil.ReadDir(path.Join(s.config.Home, "remote-log"))
	if err != nil {
		log.Error("read dir: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read dir")
	}
	logs := []string{}
	for _, l := range ls {
		logs = append(logs, l.Name())
	}

	entities, total, err := s.model.ListLogFiles(logs, page, size)
	if err != nil {
		return nil, err
	}
	return utils.NewPaginationResponse(total, entities), nil
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

func (s *Server) importLog(r *http.Request) (*model.LogEntity, error) {
	var err error
	inspectionId, err := s.upload(r)
	if err != nil {
		return nil, err
	}

	err = s.unpack(inspectionId)
	if err != nil {
		log.Error("unpack: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on unpack file")
	}

	inspection := &model.Inspection{
		Uuid:   inspectionId,
		Status: "running",
	}
	err = s.model.SetInspection(inspection)
	if err != nil {
		log.Error("create inspection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	err = s.analyze(inspectionId)
	if err != nil {
		log.Error("analyze ", inspectionId, ": ", err)
		inspection.Status = "exception"
		inspection.Message = "analyze failed"
		s.model.SetInspection(inspection)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on import log")
	}

	inspection, err = s.model.GetInspectionDetail(inspectionId)
	if err != nil {
		log.Error("get inspection detail:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_QUERY_ERROR", "error on query data")
	}
	if inspection == nil {
		log.Error("not found inspection after import log")
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "inspection not found")
	}

	return &model.LogEntity{Id: inspection.Uuid, InstanceName: inspection.InstanceName}, nil
}

func (s *Server) collectLog(instanceId, inspectionId string, begin, end time.Time) error {
	cmd := exec.Command(
		s.config.Collector,
		fmt.Sprintf("--instance-id=%s", instanceId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--topology=%s", path.Join(s.config.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(s.config.Home, "inspection")),
		"--collect=log",
		fmt.Sprintf("--log-dir=%s", path.Join(s.config.Home, "remote-log", instanceId)),
		fmt.Sprintf("--log-spliter=%s", s.config.Spliter),
		fmt.Sprintf("--begin=%s", begin.Format(time.RFC3339)),
		fmt.Sprintf("--end=%s", end.Format(time.RFC3339)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("run ", s.config.Collector, ": ", err)
		return err
	}
	return nil
}

func (s *Server) exportLog(w http.ResponseWriter, r *http.Request) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	begin := time.Now().Add(time.Duration(-1) * time.Hour)
	end := time.Now()

	if err := s.collectLog(instanceId, inspectionId, begin, end); err != nil {
		log.Error("collect log:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := s.pack(inspectionId); err != nil {
		log.Error("pack: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	localFile, err := os.Open(path.Join(s.config.Home, "package", inspectionId+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+inspectionId+".tar.gz")
	io.Copy(w, localFile)
}

func (s *Server) uploadLog(ctx context.Context, r *http.Request) (*utils.SimpleResponse, error) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	begin := time.Now().Add(time.Duration(-1) * time.Hour)
	end := time.Now()

	if err := s.collectLog(instanceId, inspectionId, begin, end); err != nil {
		log.Error("collect log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on collect log")
	}

	if err := s.pack(inspectionId); err != nil {
		log.Error("pack: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on pack log")
	}

	localFile, err := os.Open(path.Join(s.config.Home, "package", inspectionId+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read file")
	}
	defer localFile.Close()

	if err := os.Setenv("AWS_ACCESS_KEY_ID", s.config.Aws.AccessKey); err != nil {
		log.Error("set env: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ENV_ERROR", "error on set env")
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", s.config.Aws.AccessSecret); err != nil {
		log.Error("set env: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ENV_ERROR", "error on set env")
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(s.config.Aws.Region),
	}))
	service := s3.New(sess)

	_, err = service.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.config.Aws.Bucket),
		Key:    aws.String(s.config.User.Name + "/logs/" + inspectionId + ".tar.gz"),
		Body:   localFile,
	})
	if err != nil {
		log.Error("upload: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on upload")
	}

	return nil, nil
}
