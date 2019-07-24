package server

import (
	"net/http"
)

func (s *Server) static(prefix, root string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(http.Dir(root)))
}
