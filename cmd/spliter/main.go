package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/pingcap/tidb-foresight/version"
)

func main() {
	printVersion := flag.Bool("V", false, "print version info")
	src := flag.String("src", "", "source directory")
	dst := flag.String("dst", "", "target directory")
	begin := flag.String("begin", time.Now().AddDate(0, 0, -1).Format(time.RFC3339), "the begin of the log time (RFC3339)")
	end := flag.String("end", time.Now().AddDate(0, 0, 0).Format(time.RFC3339), "the end of the log time (RFC3339)")

	flag.Parse()

	if *printVersion {
		version.PrintReleaseInfo()
		return
	}

	if *src == "" || *dst == "" {
		fmt.Println("both src and dst must be specifed")
		os.Exit(1)
	}

	from, err := time.Parse(time.RFC3339, *begin)
	if err != nil {
		fmt.Println("parse begin:", err)
		os.Exit(1)
	}

	to, err := time.Parse(time.RFC3339, *end)
	if err != nil {
		fmt.Println("parse end:", err)
		os.Exit(1)
	}

	err = copy(*src, *dst, from, to)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
