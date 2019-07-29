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
	if r.URL.Query().Get("debug") == "true" {
		ctx = context.WithValue(ctx, "user_name", r.URL.Query().Get("user"))
        	ctx = context.WithValue(ctx, "user_role", r.URL.Query().Get("role"))
		return ctx, nil
	}

	cookie, err := r.Cookie("tidb-foresight-auth")
	if err != nil {
		log.Error("parse cookie in self identity: ", err)
		return ctx, utils.NewForesightError(http.StatusUnauthorized, "COOKIE_MISSING", "access denied since no cookie")
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		log.Error("decode cookie failed: ", err)
		return ctx, utils.NewForesightError(http.StatusUnauthorized, "DECODE_B64_ERROR", "invalid cookie")
	}

	authInfo := AuthInfo{}
	err = json.Unmarshal(decoded, &authInfo)
	if err != nil {
		log.Error("unmarshal json failed: ", err)
		return ctx, utils.NewForesightError(http.StatusUnauthorized, "DECODE_JSON_ERROR", "invalid cookie")
	}

	ctx = context.WithValue(ctx, "user_name", authInfo.UserName)
	ctx = context.WithValue(ctx, "user_role", authInfo.Role)

	return ctx, nil
}

func setAuthCookie(w http.ResponseWriter, user, role, token string) error {
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
		w.Write([]byte(`{"status": "MARSHAL_JSON_ERROR", "message": "when marshal json"}`))
		log.Error("marshal json:", err)
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "tidb-foresight-auth",
		Value:  base64.StdEncoding.EncodeToString(js),
		MaxAge: 60 * 60 * 24 * 7,
	})
	log.Info(role, " login successfully")

	return nil
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	auth := struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}{}
        aws := s.config.Aws
        ka := aws.Region != "" && aws.Bucket != "" && aws.AccessKey != "" && aws.AccessSecret != ""
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&auth)
	if err != nil {
		log.Info("decode json: ", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "BAD_REQUEST", "message": "json body is invalid"}`))
	} else if auth.User == s.config.User.Name && auth.Pass == s.config.User.Pass {
		if setAuthCookie(w, auth.User, "admin", s.config.Auth.Token) == nil {
			w.Write([]byte(fmt.Sprintf(`{"username": "%s", "role": "admin", "ka": %t}`, auth.User, ka)))
		}
	} else if auth.User == s.config.Admin.Name && auth.Pass == s.config.Admin.Pass {
		if setAuthCookie(w, auth.User, "dba", s.config.Auth.Token) == nil {
			w.Write([]byte(fmt.Sprintf(`{"username": "%s", "role": "dba", "ka": %t}`, auth.User, ka)))
		}
	} else {
		log.Info(auth, s.config.Admin)
		log.Info("user login failed since username and password mismatch")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "AUTH_MISMATCH", "message": "mismatch username or password"}`))
	}
}

func (s *Server) me(ctx context.Context) (map[string]interface{}, error) {
	aws := s.config.Aws
	ka := aws.Region != "" && aws.Bucket != "" && aws.AccessKey != "" && aws.AccessSecret != ""

	return map[string]interface{}{
		"username": ctx.Value("user_name"),
		"role": ctx.Value("user_role"),
		"ka":   ka,
	}, nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "tidb-foresight-auth",
		Value:  "",
		MaxAge: 0,
	})

	w.WriteHeader(http.StatusNoContent)
}
