package auth

import (
	"net/http"
)

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "tidb-foresight-auth",
		Value:  "",
		MaxAge: 0,
	})

	w.WriteHeader(http.StatusNoContent)
}
