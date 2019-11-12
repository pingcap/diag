/// This package will not be used as handler. Instead, it is designed as a
/// helper for handlers. It unify the handling of paging and some other http handling.

package utils

import (
	"encoding/json"
	"fmt"
	"github.com/pingcap/tidb-foresight/utils"
	"github.com/pingcap/tidb-foresight/wrapper/db"
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
func LoadRouterVar(r *http.Request, route string) string {
	s := LoadSelectableRouterVar(r, route)
	if s == nil {
		panic(fmt.Sprintf("%s in LoadRouterVar not exists", route))
	}
	return *s
}

// Load from route. The route must exists, otherwise the program will panic.
func LoadSelectableRouterVar(r *http.Request, route string) *string {
	if v, ok := mux.Vars(r)[route]; !ok {
		return nil
	} else {
		return &v
	}
}

// Load from body of http and parse it into body. The body is intend to be
// a map or a struct for the required response.
// If this function return an error, the caller should better returns an `utils.ParamsMismatch`.
func LoadJsonFromHttpBody(r *http.Request, body interface{}) error {
	err := json.NewDecoder(r.Body).Decode(body)
	return err
}

func GormErrorMapper(err error, originError utils.StatusError) utils.StatusError {
	if db.IsNotFound(err) {
		return utils.TargetObjectNotFound
	} else {
		return originError
	}
}
