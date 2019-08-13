package main

import (
	"flag"
	"path"

	"github.com/pingcap/tidb-foresight/api/bootstrap"
	"github.com/pingcap/tidb-foresight/api/release"
	"github.com/pingcap/tidb-foresight/api/server"
	log "github.com/sirupsen/logrus"
)

func main() {
	printVersion := flag.Bool("V", false, "print version info")
	homepath := flag.String("home", "/tmp/tidb-foresight", "tidb-foresight work home")
	address := flag.String("address", "0.0.0.0:8888", "tidb foresight listen address")
	pioneer := flag.String("pioneer", "", "tool to parse inventory.ini and get basic cluster info")
	collector := flag.String("collector", "", "tool to collect cluster info")
	analyzer := flag.String("analyzer", "", "tool to analyze cluster info")
	spliter := flag.String("spliter", "", "tool to split logs")
	syncer := flag.String("syncer", "", "tool to sync logs from cluster")

	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if *printVersion {
		release.PrintReleaseInfo()
		return
	}

	config, db := bootstrap.MustInit(*homepath)
	defer db.Close()

	config.Home = *homepath
	config.Address = *address
	config.Pioneer = *pioneer
	config.Collector = *collector
	config.Analyzer = *analyzer
	config.Spliter = *spliter
	config.Syncer = *syncer

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
	if config.Syncer == "" {
		config.Syncer = path.Join(config.Home, "bin", "syncer")
	}

	s := server.NewServer(config, db)

	log.Panic(s.Run())
}
