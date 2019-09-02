package bootstrap

import (
	"os"
	"path"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

const defaultConfig = `# This is the default config file generated by tidb-foresight

[auth]
token = "tidb"

[user]
name = "foresight"
password = "foresight"

[admin]
name = "admin"
password = "admin"

[aws]
region = ""
bucket = ""
aws_access_key_id = ""
aws_secret_access_key = ""

[log]
interval = 300		# 5min
bwlimit = 10		# KB/s
threshold = 100		# GB

[influxdb]
endpoint = "http://127.0.0.1:9528"

[prometheus]
endpoint = "http://127.0.0.1:9529"
`

type ForesightConfig struct {
	Home      string
	Address   string
	Pioneer   string
	Collector string
	Analyzer  string
	Spliter   string
	Syncer    string

	Auth struct {
		Token string `toml:"token"`
	} `toml:"auth"`

	User struct {
		Name string `toml:"name"`
		Pass string `toml:"password"`
	} `toml:"user"`

	Admin struct {
		Name string `toml:"name"`
		Pass string `toml:"password"`
	} `toml:"admin"`

	Aws struct {
		Region       string `toml:"region"`
		Bucket       string `toml:"bucket"`
		AccessKey    string `toml:"aws_access_key_id"`
		AccessSecret string `toml:"aws_secret_access_key"`
	}

	Log struct {
		Interval  int `toml:"interval`
		Bwlimit   int `toml:"bwlimit`
		Threshold int `toml:"threshold"`
	} `toml:"log"`

	Influx struct {
		Endpoint string `toml:"endpoint"`
	} `toml:"influxdb"`

	Prometheus struct {
		Endpoint string `toml:"endpoint"`
	} `toml:"prometheus"`
}

func initConfig(home string) *ForesightConfig {
	fpath := path.Join(home, "tidb-foresight.toml")
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Warn("config file ", fpath, " not found, a default one will be generated")

		f, err := os.Create(fpath)
		if err != nil {
			log.Panic("create default config file failed: ", err)
		}
		defer f.Close()

		if _, err = f.WriteString(defaultConfig); err != nil {
			log.Panic("error occurs while generate default config: ", err)
		}

		if err = f.Sync(); err != nil {
			log.Panic("error occurs while sync default config: ", err)
		}
	}

	config := &ForesightConfig{
		Home:      home,
		Pioneer:   path.Join(home, "bin", "pioneer"),
		Collector: path.Join(home, "bin", "collector"),
		Analyzer:  path.Join(home, "bin", "analyzer"),
		Spliter:   path.Join(home, "bin", "spliter"),
		Syncer:    path.Join(home, "bin", "syncer"),
	}
	if _, err := toml.DecodeFile(fpath, config); err != nil {
		log.Panic("parse config failed: ", err)
	}

	return config
}
