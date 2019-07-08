package server

import (
	"net/http"
	"testing"
	"github.com/pingcap/tidb-foresight/bootstrap"
)

func TestLogin(t *testing.T) {
	req, err := http.NewRequest("GET", "http://whatever.com", nil)
	if err != nil {
		t.Error(err)
	}
	q := req.URL.Query()
	q.Add("username", "user")
	q.Add("password", "pass")
	req.URL.RawQuery = q.Encode()

	NewServer(&bootstrap.ForesightConfig{
		Home: "/tmp",
	}, nil)

	type M struct {}

	func (m *M) test() {}
	//s.login()
}

func TestLogout(t *testing.T) {

}