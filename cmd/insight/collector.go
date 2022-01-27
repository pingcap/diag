// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/pingcap/diag/collector/sysinfo"
)

func parseOpts() sysinfo.Options {
	optProc := flag.Bool("proc", false, "Also collect process list.")
	optSyscfg := flag.Bool("syscfg", false, "Also collect system configs.")
	optDmesg := flag.Bool("dmesg", false, "Also collect kernel logs.")
	flag.Parse()

	opts := sysinfo.Options{
		Proc:   *optProc,
		Syscfg: *optSyscfg,
		Dmesg:  *optDmesg,
	}
	return opts
}

func main() {
	opts := parseOpts()

	var info sysinfo.InsightInfo
	info.GetInfo(opts)

	data, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(data))
}
