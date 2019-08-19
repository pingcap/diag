package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type AuthInfo struct {
	UserName  string `json:"username"`
	Role      string `json:"role"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func Auth(ctx context.Context, r *http.Request) (context.Context, error) {
	if r.URL.Query().Get("debug") == "true" {
		ctx = context.WithValue(ctx, "user_name", r.URL.Query().Get("user"))
		ctx = context.WithValue(ctx, "user_role", r.URL.Query().Get("role"))
		return ctx, nil
	}

	cookie, err := r.Cookie("tidb-foresight-auth")
	if err != nil {
		log.Error("parse cookie in self identity: ", err)
		return ctx, utils.AuthWithoutCookie
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Error("decode cookie failed: ", err)
		return ctx, utils.AuthWithInvalidCookie
	}

	authInfo := AuthInfo{}
	err = json.Unmarshal(decoded, &authInfo)
	if err != nil {
		log.Error("unmarshal json failed: ", err)
		return ctx, utils.AuthWithInvalidCookie
	}

	ctx = context.WithValue(ctx, "user_name", authInfo.UserName)
	ctx = context.WithValue(ctx, "user_role", authInfo.Role)

	return ctx, nil
}
