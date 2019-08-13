package main

import (
	"flag"

	"github.com/pingcap/tidb-foresight/api/bootstrap"
	"github.com/pingcap/tidb-foresight/api/server"
	"github.com/pingcap/tidb-foresight/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	printVersion := flag.Bool("V", false, "print version info")
	homepath := flag.String("home", "/tmp/tidb-foresight", "tidb-foresight work home")
	address := flag.String("address", "0.0.0.0:8888", "tidb foresight listen address")

	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if *printVersion {
		version.PrintReleaseInfo()
		return
	}

	config, db := bootstrap.MustInit(*homepath, *address)
	defer db.Close()

	s := server.NewServer(config, db)

	log.Panic(s.Run())
}
