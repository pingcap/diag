package server

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
)

const MAX_FILE_SIZE = 32 * 1024 * 1024

func (s *Server) static(prefix, root string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(root)))
}

// TODO: response client with json content
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	log.Info("uploading file...")

	r.ParseMultipartForm(MAX_FILE_SIZE)

	// FormFile returns the first file for the given key `file`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("file")
	if err != nil {
		log.Error("error retrieving the file: ", err)
		return
	}
	defer file.Close()
	log.Infof("Uploaded File: %+v", handler.Filename)
	log.Infof("File Size: %+v", handler.Size)
	log.Infof("MIME Header: %+v", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	localFile, err := os.Create(path.Join(s.home, "static", handler.Filename))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error(err)
		return
	}
	defer localFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Error(err)
		return
	}
	// write this byte array to our temporary file
	if _, err := localFile.Write(fileBytes); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	w.WriteHeader(http.StatusOK)
	log.Info("upload successfully")
}
