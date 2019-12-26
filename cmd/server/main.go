package main

import (
	"flag"

	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/server"
	"github.com/pingcap/tidb-foresight/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	printVersion := flag.Bool("V", false, "print version info")
	homepath := flag.String("home", "/tmp/tidb-foresight", "tidb-foresight work home")
	address := flag.String("address", "0.0.0.0:9527", "tidb foresight listen address")

	prometheusAddress := flag.String("prometheus", "127.0.0.1:9529", "prometheus listen address")
	influxdbAddress := flag.String("influxdb", "127.0.0.1:9528", "influxdb listen address")

	debug := flag.Bool("debug", false, "if using debug logging")

	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,

	})

	if *printVersion {
		version.PrintReleaseInfo()
		return
	}

	config, db := bootstrap.MustInit(*homepath)

	config.Influx.Endpoint = *influxdbAddress
	config.Prometheus.Endpoint = *prometheusAddress

	defer db.Close()

	s := server.New(config, db)

	log.Panic(s.Run(*address))
}
