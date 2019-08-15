package auth

import (
	"context"
	"net/http"

	"github.com/pingcap/fn"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/utils"
)

type selfHandler struct {
	c *bootstrap.ForesightConfig
}

func Me(c *bootstrap.ForesightConfig) http.Handler {
	return &selfHandler{c}
}

func (h *selfHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fn.Wrap(h.me).ServeHTTP(w, r)
}

func (h *selfHandler) me(ctx context.Context) (map[string]interface{}, utils.StatusError) {
	aws := h.c.Aws
	ka := aws.Region != "" && aws.Bucket != "" && aws.AccessKey != "" && aws.AccessSecret != ""

	return map[string]interface{}{
		"username": ctx.Value("user_name"),
		"role":     ctx.Value("user_role"),
		"ka":       ka,
	}, nil
}
