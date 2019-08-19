package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"

	log "github.com/sirupsen/logrus"
)

const MAX_FILE_SIZE = 32 * 1024 * 1024

func PackInspection(home, inspectionId string) error {
	cmd := exec.Command(
		"tar",
		"-czvf",
		path.Join(home, "package", inspectionId+".tar.gz"),
		"-C",
		path.Join(home, "inspection"),
		inspectionId,
	)
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run tar: ", err)
		return err
	}
	return nil
}

func UnpackInspection(home, inspectionId string) error {
	cmd := exec.Command(
		"tar",
		"-xzvf",
		path.Join(home, "package", inspectionId+".tar.gz"),
		"-C",
		path.Join(home, "inspection"),
	)
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run tar: ", err)
		return err
	}
	return nil
}

func UploadInspection(home string, r *http.Request) (string, StatusError) {
	log.Info("upload inspection")

	r.ParseMultipartForm(MAX_FILE_SIZE)

	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Error("retrieving file: ", err)
		return "", InvalidFile
	}
	defer file.Close()
	log.Infof("file name: %+v", handler.Filename)
	log.Infof("file size: %+v", handler.Size)
	log.Infof("mime header: %+v", handler.Header)

	re := regexp.MustCompile("^([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}).tar.gz$")
	if !re.MatchString(handler.Filename) {
		return "", InvalidFile
	}
	ms := re.FindStringSubmatch(handler.Filename)
	inspectionId := ms[1]

	localFile, err := os.Create(path.Join(home, "package", handler.Filename))
	if err != nil {
		log.Error("create file: ", err)
		return "", FileOpError
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, file)
	if err != nil {
		log.Error("write file", err)
		return "", FileOpError
	}

	log.Info("upload successfully")
	return inspectionId, nil
}

func Analyze(analyzer, home, influx, prom, inspectionId string) error {
	cmd := exec.Command(
		analyzer,
		fmt.Sprintf("--home=%s", home),
		fmt.Sprintf("--inspection-id=%s", inspectionId),
	)
	cmd.Env = append(
		os.Environ(),
		"INFLUX_ADDR="+influx,
		"PROM_ADDR="+prom,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Info(cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Error("run ", analyzer, ": ", err)
		return err
	}
	return nil
}
