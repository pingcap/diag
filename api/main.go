package main

import (
	"flag"

	log "github.com/sirupsen/logrus"
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/server"
)

func main() {
	homepath := flag.String("home", "/tmp/tidb-foresight", "tidb-foresight work home")
	address := flag.String("address", "0.0.0.0:8888", "tidb foresight listen address")

	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})

	config, db := bootstrap.MustInit(*homepath)
	defer db.Close()

	config.Home = *homepath
	config.Address = *address

	s := server.NewServer(config, db)

	log.Error(s.Run())
}