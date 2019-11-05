package utils

import (
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
)

func SaveFile(r io.Reader, targetPath string) error {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		log.Info("read source error ", err)
		log.Error("read source: ", err)
		return err
	}

	localFile, err := os.Create(targetPath)
	if err != nil {
		log.Info("create local file error: ", err)
		log.Error("create local file: ", err)
		return err
	}
	defer localFile.Close()

	_, err = localFile.Write(content)
	if err != nil {
		log.Info("write local file error: ", err)
		log.Error("write local file: ", err)
		return err
	}

	return nil
}
