package inspection

import (
	"context"
	"net/http"
	"os"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type uploadInspectionHandler struct {
	c *bootstrap.ForesightConfig
}

func UploadInspection(c *bootstrap.ForesightConfig) http.Handler {
	return &uploadInspectionHandler{c}
}

func (h *uploadInspectionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.uploadInspection).ServeHTTP(w, r)
}

func (h *uploadInspectionHandler) uploadInspection(ctx context.Context, r *http.Request) (*utils.SimpleResponse, utils.StatusError) {
	uuid := mux.Vars(r)["id"]

	if _, err := os.Stat(path.Join(h.c.Home, "package", uuid+".tar.gz")); os.IsNotExist(err) {
		err = utils.PackInspection(h.c.Home, uuid)
		if err != nil {
			log.Error("pack: ", err)
			return nil, utils.FileOpError
		}
	}

	localFile, err := os.Open(path.Join(h.c.Home, "package", uuid+".tar.gz"))
	if err != nil {
		log.Error("read file: ", err)
		return nil, utils.FileOpError
	}
	defer localFile.Close()

	if err := os.Setenv("AWS_ACCESS_KEY_ID", h.c.Aws.AccessKey); err != nil {
		log.Error("set env: ", err)
		return nil, utils.SystemOpError
	}
	if err := os.Setenv("AWS_SECRET_ACCESS_KEY", h.c.Aws.AccessSecret); err != nil {
		log.Error("set env: ", err)
		return nil, utils.SystemOpError
	}

	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(h.c.Aws.Region),
	}))
	service := s3.New(sess)

	_, err = service.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(h.c.Aws.Bucket),
		Key:    aws.String(h.c.User.Name + "/inspections/" + uuid + ".tar.gz"),
		Body:   localFile,
	})
	if err != nil {
		log.Error("upload: ", err)
		return nil, utils.NetworkError
	}

	return nil, nil
}
