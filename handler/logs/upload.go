package logs

import (
	"context"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type uploadLogHandler struct {
	c *bootstrap.ForesightConfig
}

func UploadLog(c *bootstrap.ForesightConfig) http.Handler {
	return &uploadLogHandler{c}
}

func (h *uploadLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.uploadLog).ServeHTTP(w, r)
}

func (h *uploadLogHandler) uploadLog(ctx context.Context, r *http.Request) (*utils.SimpleResponse, utils.StatusError) {
	instanceId := mux.Vars(r)["id"]
	inspectionId := uuid.New().String()
	begin := time.Now().Add(time.Duration(-1) * time.Hour)
	end := time.Now()

	if bt, e := time.Parse(time.RFC3339, r.URL.Query().Get("begin")); e == nil {
		begin = bt
	}
	if et, e := time.Parse(time.RFC3339, r.URL.Query().Get("end")); e == nil {
		end = et
	}

	if err := utils.CollectLog(
		h.c.Collector, h.c.Home, h.c.User.Name, instanceId, inspectionId, begin, end,
	); err != nil {
		log.Error("collect log:", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on collect log")
	}

	if err := utils.PackInspection(h.c.Home, inspectionId); err != nil {
		log.Error("pack: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on pack log")
	}

	localFile, err := os.Open(path.Join(h.c.Home, "package", inspectionId+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_FS_ERROR", "error on read file")
	}
	defer localFile.Close()

	if err := os.Setenv("AWS_ACCESS_KEY_ID", h.c.Aws.AccessKey); err != nil {
		log.Error("set env: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ENV_ERROR", "error on set env")
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", h.c.Aws.AccessSecret); err != nil {
		log.Error("set env: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ENV_ERROR", "error on set env")
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(h.c.Aws.Region),
	}))
	service := s3.New(sess)

	_, err = service.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(h.c.Aws.Bucket),
		Key:    aws.String(h.c.User.Name + "/logs/" + inspectionId + ".tar.gz"),
		Body:   localFile,
	})
	if err != nil {
		log.Error("upload: ", err)
		return nil, utils.NewForesightError(http.StatusInternalServerError, "SERVER_ERROR", "error on upload")
	}

	return nil, nil
}
