/// This file is created for currently unimplemented interface.
/// It will be deleted after the whole logic is implemented.
package emphasis

import (
	"github.com/pingcap/tidb-foresight/bootstrap"
	"net/http"
)

type unimplementedHandler struct {
	c *bootstrap.ForesightConfig
}

func Unimplemented(c *bootstrap.ForesightConfig) http.Handler {
	return &unimplementedHandler{c}
}

func (h *unimplementedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.unimplementedHandler(w, r)
}

func (h *unimplementedHandler) unimplementedHandler(w http.ResponseWriter, r *http.Request) {
	return
}
