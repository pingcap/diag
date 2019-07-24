package bootstrap

import (
	"io/ioutil"
	"os"
	"testing"
)

const customConfig = `[sched]
auto = true
duration = "1d"

[auth]
token = "tidb"

[user]
name = "custom-user"
password = "custom-pass"

[admin]
name = "admin"
password = "admin"

[log]
interval = 600		# 5min
bwlimit = 10		# KB/s
`

func TestDefaultConfigAutoGen(t *testing.T) {
	path := "/tmp/test-default-config-auto-gen"

	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			t.Error(err)
		}
	}

	config := initConfig(path)
	if config == nil {
		t.Error("nil config returned")
	}

	if _, err := os.Stat(path); err != nil {
		t.Error(err)
	}

	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	if string(bytes) != defaultConfig {
		t.Error("default config mismatch")
	}

	if config.Sched.Auto != false {
		t.Error("default auto schedule is not false")
	}

	if config.Sched.Duration != "1h" {
		t.Error("default schedule duration should be 1h")
	}

	if config.Log.Interval != 600 {
		t.Error("default log collect interval should be 600(seconds)")
	}

	if config.Log.Bwlimit != 10 {
		t.Error("default log collect bwlimit should be 10(kb/s)")
	}
}

func TestCustomConfig(t *testing.T) {
	path := "/tmp/test-custom-config"

	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			t.Error(err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Error(err)
	}
	defer f.Close()

	if _, err = f.WriteString(customConfig); err != nil {
		t.Error(err)
	}

	if err = f.Sync(); err != nil {
		t.Error(err)
	}

	config := initConfig(path)
	if config == nil {
		t.Error("nil config returned")
	}

	if config.Sched.Auto != true {
		t.Error("custom auto schedule is not true")
	}

	if config.Sched.Duration != "1d" {
		t.Error("custom schedule duration should be 1d")
	}

	if config.User.Name != "custom-user" {
		t.Error("user name should be custom-user")
	}

	if config.User.Pass != "custom-pass" {
		t.Error("user pass should be custom-pass")
	}
}
