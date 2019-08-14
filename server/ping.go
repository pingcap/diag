package server

import (
	"net/http"
)

type T struct{}

func (s *Server) ping(r *http.Request) (*T, error) {
	return nil, nil
}
