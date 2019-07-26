package server

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/tidb-foresight/model"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
)

func (s *Server) collect(instanceId, inspectionId string) error {
	config, err := s.model.GetInstanceConfig(instanceId)
	if err != nil {
		log.Error("get instance config: ", err)
		return err
	}
	items := []string{"metric","basic","dbinfo","config","profile"}
	if config != nil {
		if config.CollectHardwareInfo {
			items = append(items, "hardware")
		}
		if config.CollectSoftwareInfo {
			items = append(items, "software")
		}
		if config.CollectLog {
			items = append(items, "log")
		}
		if config.CollectDemsg {
			items = append(items, "demsg")
		}
	}
	cmd := exec.Command(
		s.config.Collector,
		fmt.Sprintf("--instance-id=%s", inspectionId),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
		fmt.Sprintf("--inventory=%s", path.Join(s.config.Home, "inventory", instanceId+".ini")),
		fmt.Sprintf("--topology=%s", path.Join(s.config.Home, "topology", instanceId+".json")),
		fmt.Sprintf("--data-dir=%s", path.Join(s.config.Home, "inspection")),
		fmt.Sprintf("--collect=%s", strings.Join(items, ",")),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd)
	err = cmd.Run()
	if err != nil {
		log.Error("run ", s.config.Collector, ": ", err)
		return err
	}
	return nil
}

func (s *Server) analyze(inspectionId string) error {
	cmd := exec.Command(
		s.config.Analyzer,
		fmt.Sprintf("--db=%s", path.Join(s.config.Home, "sqlite.db")),
		fmt.Sprintf("--src=%s", path.Join(s.config.Home, "inspection", inspectionId)),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", s.config.Analyzer, ": ", err)
		return err
	}
	return nil
}

func (s *Server) pack(inspectionId string) error {
	cmd := exec.Command(
		"tar",
		"-czvf",
		path.Join(s.config.Home, "package", inspectionId+".tar.gz"),
		"-C",
		path.Join(s.config.Home, "inspection"),
		inspectionId,
	)
	log.Info(cmd)
	err := cmd.Run()
	if err != nil {
		log.Error("run tar: ", err)
		return err
	}
	return nil
}

func (s *Server) uppack(inspectionId string) error {
	cmd := exec.Command(
		"tar",
		"-xzvf",
		path.Join(s.config.Home, "package", inspectionId+".tar.gz"),
		"-C",
		path.Join(s.config.Home, "inspection"),
	)
	log.Info(cmd)
	err := cmd.Run()
	if err != nil {
		log.Error("run tar: ", err)
		return err
	}
	return nil
}

func (s *Server) upload(r *http.Request) (string, utils.StatusError) {
	log.Info("upload inspection")

	const MAX_FILE_SIZE = 32 * 1024 * 1024

	r.ParseMultipartForm(MAX_FILE_SIZE)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Error("retrieving file: ", err)
		return "", utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "error on retrieving file")
	}
	defer file.Close()
	log.Infof("file name: %+v", handler.Filename)
	log.Infof("file size: %+v", handler.Size)
	log.Infof("mime header: %+v", handler.Header)

	re := regexp.MustCompile("^([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}).tar.gz$")
	if !re.MatchString(handler.Filename) {
		return "", utils.NewForesightError(http.StatusBadRequest, "BAD_REQUEST", "invalid file name")
	}
	m := re.FindStringSubmatch(handler.Filename)
	inspectionId := m[1]

	localFile, err := os.Create(path.Join(s.config.Home, "package", handler.Filename))
	if err != nil {
		log.Error("create file: ", err)
		return "", utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on write file")
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, file)
	if err != nil {
		log.Error("write file", err)
		return "", utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on upload file")
	}

	log.Info("upload successfully")
	return inspectionId, nil
}

func (s *Server) listAllInspections(r *http.Request) (*utils.PaginationResponse, error) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}
	inspections, total, err := s.model.ListAllInspections(page, size)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}
	return utils.NewPaginationResponse(total, inspections), nil
}

func (s *Server) listInspections(r *http.Request) (*utils.PaginationResponse, error) {
	instanceId := mux.Vars(r)["id"]
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err := strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}

	inspections, total, err := s.model.ListInspections(instanceId, page, size)
	if err != nil {
		log.Error("list inspections: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_SELECT_ERROR", "error on query database")
	}

	return utils.NewPaginationResponse(total, inspections), nil
}

func (s *Server) createInspection(r *http.Request) (*model.Inspection, error) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()

	inspection := &model.Inspection{
		Uuid:       inspectionId,
		InstanceId: instanceId,
		Status:     "running",
		Type:       "manual",
	}
	err := s.model.SetInspection(inspection)
	if err != nil {
		log.Error("set inpsection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_INSERT_ERROR", "error on insert data")
	}

	go func() {
		err := s.collect(instanceId, inspectionId)
		if err != nil {
			log.Error("collect ", inspectionId, ": ", err)
			return
		}
		err = s.analyze(inspectionId)
		if err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			return
		}
	}()

	return inspection, nil
}

func (s *Server) exportInspection(w http.ResponseWriter, r *http.Request) {
	uuid := mux.Vars(r)["id"]

	if _, err := os.Stat(path.Join(s.config.Home, "package", uuid+".tar.gz")); os.IsNotExist(err) {
		err = s.pack(uuid)
		if err != nil {
			log.Error("pack: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	localFile, err := os.Open(path.Join(s.config.Home, "package", uuid+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer localFile.Close()

	io.Copy(w, localFile)
}

func (s *Server) importInspection(r *http.Request) (*model.Inspection, error) {
	var err error
	inspectionId, err := s.upload(r)
	if err != nil {
		return nil, err
	}

	err = s.uppack(inspectionId)
	if err != nil {
		log.Error("unpack: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on uppack file")
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

	go func() {
		err = s.analyze(inspectionId)
		if err != nil {
			log.Error("analyze ", inspectionId, ": ", err)
			return
		}
	}()

	return inspection, nil
}

func (s *Server) uploadInspection(ctx context.Context, r *http.Request) (*utils.SimpleResponse, error) {
	uuid := mux.Vars(r)["id"]

	if _, err := os.Stat(path.Join(s.config.Home, "package", uuid+".tar.gz")); os.IsNotExist(err) {
		err = s.pack(uuid)
		if err != nil {
			log.Error("pack: ", err)
			return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on pack file")
		}
	}

	localFile, err := os.Open(path.Join(s.config.Home, "package", uuid+".tar.gz"))
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
		Key:    aws.String(s.config.User.Name + "/" + uuid + ".tar.gz"),
		Body:   localFile,
	})
	if err != nil {
		log.Error("upload: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on upload")
	}

	return utils.NewSimpleResponse("OK", "success"), nil
}

func (s *Server) deleteInspection(r *http.Request) (*utils.SimpleResponse, error) {
	uuid := mux.Vars(r)["id"]

	if err := s.model.DeleteInspection(uuid); err != nil {
		log.Error("delete inspection: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "DB_DELETE_ERROR", "error on delete data")
	}

	return utils.NewSimpleResponse("OK", "success"), nil
}
