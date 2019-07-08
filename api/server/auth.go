package server

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pingcap/tidb-foresight/utils"

	log "github.com/sirupsen/logrus"
)

type AuthInfo struct {
	UserName  string `json:"username"`
	Role      string `json:"role"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func (s *Server) auth(ctx context.Context, r *http.Request) (context.Context, error) {
	cookie, err := r.Cookie("tidb-foresight-auth")
	if err != nil {
		log.Error("parse cookie in self identity: ", err)
		return ctx, utils.NewForesightError(http.StatusForbidden, "COOKIE_MISSING", "用户Cookie缺失，禁止访问")
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Error("decode cookie failed: ", err)
		return ctx, utils.NewForesightError(http.StatusBadRequest, "DECODE_B64_ERROR", "无法解析的Cookie")
	}

	authInfo := AuthInfo{}
	err = json.Unmarshal(decoded, &authInfo)
	if err != nil {
		log.Error("unmarshal json failed: ", err)
		return ctx, utils.NewForesightError(http.StatusBadRequest, "DECODE_JSON_ERROR", "无法解析的Cookie")
	}

	ctx = context.WithValue(ctx, "user_name", authInfo.UserName)
	ctx = context.WithValue(ctx, "user_role", authInfo.Role)

	return ctx, nil
}

func setAuthCookie(w http.ResponseWriter, user, role, token string) {
	t := time.Now()
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("%s%s%d%s", user, role, t.Unix(), token)))
	bs := h.Sum(nil)
	sig := fmt.Sprintf("%x", bs)

	js, err := json.Marshal(AuthInfo{
		UserName:  user,
		Role:      role,
		Timestamp: t.Unix(),
		Signature: sig,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "MARSHAL_JSON_ERROR", "message": "序列化json时发生错误"}`))
		log.Error("序列化json时发生错误", err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "tidb-foresight-auth",
		Value:  base64.StdEncoding.EncodeToString(js),
		MaxAge: 60 * 60 * 24 * 7,
	})
	log.Info(role, " login successfully")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "OK", "message": "success"}`))
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	usernames, ok := r.URL.Query()["username"]
	if !ok || len(usernames) == 0 {
		log.Info("user login failed since without username")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "USERNAME_MISSING", "message": "用户名缺失"}`))
		return
	}
	username := usernames[0]

	passwords, ok := r.URL.Query()["password"]
	if !ok || len(passwords) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "PASSWORD_MISSING", "message": "密码缺失"}`))
		return
	}
	password := passwords[0]

	if username == s.config.User.Name && password == s.config.User.Pass {
		setAuthCookie(w, username, "user", s.config.Auth.Token)
	} else if username == s.config.Admin.Name && password == s.config.Admin.Pass {
		setAuthCookie(w, username, "admin", s.config.Auth.Token)
	} else {
		log.Info("user login failed since username and password mismatch")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "AUTH_MISMATCH", "message": "用户名和密码不匹配"}`))
	}
}

func (s *Server) me(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"user": ctx.Value("user_name").(string),
		"role": ctx.Value("user_role").(string),
	}, nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "tidb-foresight-auth",
		Value:  "",
		MaxAge: 0,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "OK", "message": "success"}`))
}
