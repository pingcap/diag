package main

import (
	"io"
	"os"
	"path"
	"time"

	"github.com/pingcap/tidb-foresight/api/logparser"
	log "github.com/sirupsen/logrus"
)

func copy(src, dest string, begin, end time.Time) error {
	files, err := logparser.ResolveDir(src)
	if err != nil {
		return err
	}
	for _, fw := range files {
		iter, err := logparser.NewIterator(fw, begin, end)
		if err != nil {
			if err != io.EOF {
				log.Warnf("create log iterator err: %s", err)
			}
			continue
		}
		f, err := createFile(dest, fw)
		if err != nil {
			return err
		}
		err = copyToFile(f, iter)
		if err != nil {
			return err
		}
	}
	return nil
}

func createFile(dest string, fw *logparser.FileWrapper) (*os.File, error) {
	dest = path.Join(dest, fw.Host, fw.Folder)
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return os.Create(path.Join(dest, fw.Filename))
}

func copyToFile(f *os.File, iterator logparser.Iterator) error {
	defer f.Close()
	for {
		item, err := iterator.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		content := item.GetContent()
		_, err = f.Write(content)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("\n"))
		if err != nil {
			return err
		}
	}
	return nil
}
