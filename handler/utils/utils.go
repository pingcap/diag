/// This package will not be used as handler. Instead, it is designed as a
/// helper for handlers. It unify the handling of paging and some other http handling.

package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// This function is used to help paging.
func LoadHttpPaging(r *http.Request) (page int64, size int64) {
	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 32)
	if err != nil {
		page = 1
	}
	size, err = strconv.ParseInt(r.URL.Query().Get("per_page"), 10, 32)
	if err != nil {
		size = 10
	}
	return
}

// Load from route. The route must exists, otherwise the program will panic.
func LoadRouterVar(r *http.Request, route string) (v string) {
	if v, ok := mux.Vars(r)[route]; !ok {
		panic(fmt.Sprintf("%s in LoadRouterVar not exists", v))
	}
	return
}

// Load from body of http and parse it into body. The body is intend to be
// a map or a struct for the required response.
// If this function return an error, the caller should better returns an `utils.ParamsMismatch`.
func LoadJsonFromHttpBody(r *http.Request, body interface{}) error {
	return json.NewDecoder(r.Body).Decode(body)
}
