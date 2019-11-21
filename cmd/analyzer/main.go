package main

import (
	"flag"

	"github.com/pingcap/tidb-foresight/analyzer"
	"github.com/pingcap/tidb-foresight/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	printVersion := flag.Bool("V", false, "print version info")
	home := flag.String("home", "/tmp/tidb-foresight", "the tidb-foresight data directory")
	inspectionId := flag.String("inspection-id", "", "the inspection to be analyze")
	flag.Parse()

	if *printVersion {
		version.PrintReleaseInfo()
		return
	}

	if *inspectionId == "" {
		log.Panic("the inspection-id must be specified")
	}

	log.Infof(`[Analyzer] Run analyzer with arguments -V %v --home %v --inspection-id %v`, printVersion, home, inspectionId)

	analyzer := analyzer.NewAnalyzer(*home, *inspectionId)
	analyzer.Run()
}
