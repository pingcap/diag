package main

import (
	"flag"
	"time"

	"github.com/pingcap/tidb-foresight/analyzer"
	"github.com/pingcap/tidb-foresight/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	beginTime := time.Now()
	defer func() {
		endTime := time.Now()
		log.Infof("Analyzer Begin at %v, end at %v, cost %v times", beginTime, endTime, endTime.Sub(beginTime))
	}()

	printVersion := flag.Bool("V", false, "print version info")
	home := flag.String("home", "/tmp/tidb-foresight", "the tidb-foresight data directory")
	inspectionId := flag.String("inspection-id", "", "the inspection to be analyze")
	flag.Parse()

	if *printVersion {
		version.PrintReleaseInfo()
		return
	}

	if *inspectionId == "" {
		*inspectionId = "555a7ce3-76f0-403a-b8c5-f9f9c5d5db2b"
		//log.Panic("the inspection-id must be specified")
	}

	log.Infof(`[Analyzer] Run analyzer with arguments -V %v --home %v --inspection-id %v`, printVersion, home, inspectionId)

	analyzer := analyzer.NewAnalyzer(*home, *inspectionId)
	analyzer.Run()
}
