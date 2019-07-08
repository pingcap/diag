package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

type AuthInfo struct {
	UserName  string `json:"username"`
	Role      string `json:"role"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

func (s *Server) authFunc(next func(http.ResponseWriter, *http.Request)) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("enter auth")
		next(w, r)
	})
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info("enter auth")
		next.ServeHTTP(w, r)
	})
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
}

/*
func checkAuthCookie(r *http.Request) bool {

}
*/

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

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("tidb-foresight-auth")
	if err != nil {
		log.Error("parse cookie in self identity: ", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Error("decode cookie failed: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status": "UKNOWN_ERROR", "message": "获取用户信息时发生未知错误"}`))
		return
	}
	log.Info(string(decoded))
}

func (s *Server) logout(http.ResponseWriter, *http.Request) {
}
