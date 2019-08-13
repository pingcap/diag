package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"time"
)

const home = "/tmp/tidb-foresight"

func main() {
	topo := flag.String("topo", path.Join(home, "topology"), "topology directory")
	target := flag.String("target", path.Join(home, "remote-log"), "target directory")
	interval := flag.Int("interval", 600, "synchronization interval (second)")
	bwlimit := flag.Int("bwlimit", 128, "bandwidth limit for synchronization (byte/s)")

	flag.Parse()

	intervalDuration := time.Duration(*interval) * time.Second
	err := Sync(*topo, *target, intervalDuration, *bwlimit)
	if err != nil {
		fmt.Println("failed to sync log:", err)
		os.Exit(1)
	}
}
