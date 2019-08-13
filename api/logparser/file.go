package logparser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type FileWrapper struct {
	Root     string
	Host     string
	Folder   string
	Filename string
}

func (fw *FileWrapper) open() (*os.File, error) {
	filePath := path.Join(fw.Root, fw.Host, fw.Folder, fw.Filename)
	return os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
}

func (fw *FileWrapper) parseFolderName() (string, string, error) {
	s := strings.Split(fw.Folder, "-")
	if len(s) < 2 {
		return "", "", fmt.Errorf("unexpect folder name: %s", s)
	}
	return s[0], s[1], nil
}

func NewFileWrapper(root, host, folder, filename string) *FileWrapper {
	return &FileWrapper{
		Root:     root,
		Host:     host,
		Folder:   folder,
		Filename: filename,
	}
}

func ResolveDir(src string) ([]*FileWrapper, error) {
	var wrappers []*FileWrapper
	dir, err := ioutil.ReadDir(src) // {cluster_uuid}
	if err != nil {
		return nil, err
	}
	for _, fi := range dir {
		host := fi.Name() // {host_ip}
		if !fi.IsDir() {
			continue
		}
		dirPath := path.Join(src, host)
		dir, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		for _, fi := range dir {
			folder := fi.Name() // {component_name}-{port}
			if !fi.IsDir() {
				continue
			}
			dirPath := path.Join(dirPath, folder)
			dir, err := ioutil.ReadDir(dirPath)
			if err != nil {
				return nil, err
			}
			for _, fi := range dir {
				filename := fi.Name()
				if fi.IsDir() {
					continue
				}
				fw := NewFileWrapper(src, host, folder, filename)
				wrappers = append(wrappers, fw)
			}
		}
	}
	return wrappers, nil
}
