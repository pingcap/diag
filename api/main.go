package main

import (
	"flag"
	"path"

	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	homepath := flag.String("home", "/tmp/tidb-foresight", "tidb-foresight work home")
	address := flag.String("address", "0.0.0.0:8888", "tidb foresight listen address")
	pioneer := flag.String("pioneer", "", "tool to parse inventory.ini and get basic cluster info")
	collector := flag.String("collector", "", "tool to collect cluster info")
	analyzer := flag.String("analyzer", "", "tool to analyze cluster info")
	spliter := flag.String("spliter", "", "tool to split logs")

	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	config, db := bootstrap.MustInit(*homepath)
	defer db.Close()

	config.Home = *homepath
	config.Address = *address
	config.Pioneer = *pioneer
	config.Collector = *collector
	config.Analyzer = *analyzer
	config.Spliter = *spliter

	if config.Pioneer == "" {
		config.Pioneer = path.Join(config.Home, "bin", "pioneer")
	}
	if config.Collector == "" {
		config.Collector = path.Join(config.Home, "bin", "collector")
	}
	if config.Analyzer == "" {
		config.Analyzer = path.Join(config.Home, "bin", "analyzer")
	}
	if config.Spliter == "" {
		config.Spliter = path.Join(config.Home, "bin", "spliter")
	}

	s := server.NewServer(config, db)

	log.Panic(s.Run())
}
