package parser

import (
	"fmt"
	"os"
	"path"
	"strings"
)

// FileWrapper is mapped from a file path, which is in the format
// of /{root}/{ip}/{component}:{port}/{xxx}.log
type FileWrapper struct {
	Root     string
	Host     string
	Folder   string
	Filename string
}

// Open the file fw represent.
func (fw *FileWrapper) Open() (*os.File, error) {
	filePath := path.Join(fw.Root, fw.Host, fw.Folder, fw.Filename)
	return os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
}

// ParseFolderName returns the component name and port it listening on.
func (fw *FileWrapper) ParseFolderName() (comp string, port string, err error) {
	s := strings.Split(fw.Folder, "-")
	if len(s) < 2 {
		return "", "", fmt.Errorf("unexpect folder name: %s", s)
	}
	return s[0], s[1], nil
}

// NewFileWrapper creates a file wrapper
func NewFileWrapper(root, host, folder, filename string) *FileWrapper {
	return &FileWrapper{
		Root:     root,
		Host:     host,
		Folder:   folder,
		Filename: filename,
	}
}

// ResolveDir traversing a folder and parse it's struct, generating
// a list of file wrapper.
func ResolveDir(src string) ([]*FileWrapper, error) {
	var wrappers []*FileWrapper
	dir, err := os.ReadDir(src) // {cluster_uuid}
	if err != nil {
		return nil, err
	}
	for _, fi := range dir {
		host := fi.Name() // {host_ip}
		if !fi.IsDir() {
			continue
		}
		dirPath := path.Join(src, host)
		dir, err := os.ReadDir(dirPath)
		if err != nil {
			return nil, err
		}
		for _, fi := range dir {
			folder := fi.Name() // {component_name}-{port}
			if !fi.IsDir() {
				continue
			}
			dirPath := path.Join(dirPath, folder)
			dir, err := os.ReadDir(dirPath)
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
