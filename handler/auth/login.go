package auth

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pingcap/tidb-foresight/bootstrap"
	log "github.com/sirupsen/logrus"
)

type loginHandler struct {
	config *bootstrap.ForesightConfig
}

func Login(config *bootstrap.ForesightConfig) http.Handler {
	return &loginHandler{config}
}

func (h *loginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := struct {
		User string `json:"username"`
		Pass string `json:"password"`
	}{}
	aws := h.config.Aws
	ka := aws.Region != "" && aws.Bucket != "" && aws.AccessKey != "" && aws.AccessSecret != ""
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&auth)
	if err != nil {
		log.Info("decode json: ", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"status": "BAD_REQUEST", "message": "json body is invalid"}`))
	} else if auth.User == h.config.User.Name && auth.Pass == h.config.User.Pass {
		if setAuthCookie(w, auth.User, "admin", h.config.Auth.Token) == nil {
			w.Write([]byte(fmt.Sprintf(`{"username": "%s", "role": "admin", "ka": %t}`, auth.User, ka)))
		}
	} else if auth.User == h.config.Admin.Name && auth.Pass == h.config.Admin.Pass {
		if setAuthCookie(w, auth.User, "dba", h.config.Auth.Token) == nil {
			w.Write([]byte(fmt.Sprintf(`{"username": "%s", "role": "dba", "ka": %t}`, auth.User, ka)))
		}
	} else {
		log.Info(auth, h.config.Admin)
		log.Info("user login failed since username and password mismatch")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status": "AUTH_MISMATCH", "message": "mismatch username or password"}`))
	}
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
