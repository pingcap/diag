package server

import (
	"net/http"
	"os"
	"path"
)

func WebServer(fs http.FileSystem) http.Handler {
	fsh := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := fs.Open(path.Clean(r.URL.Path))
		if err == nil {
			f.Close()
		}
		if os.IsNotExist(err) {
			r.URL.Path = "/"
		}
		fsh.ServeHTTP(w, r)
	})
}

func (s *Server) web(prefix, root string) http.Handler {
	return http.StripPrefix(prefix, WebServer(http.Dir(root)))
}
