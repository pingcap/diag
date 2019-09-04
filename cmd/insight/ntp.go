// Copyright © 2018 PingCAP Inc.
//
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file.
//
// Use ntpq to get basic info of NTPd on the system

package main

import (
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"
)

type TimeStat struct {
	Ver     string  `json:"version,omitempty"`
	Sync    string  `json:"sync,omitempty"`
	Stratum int     `json:"stratum,omitempty"`
	Offset  float64 `json:"offset,omitempty"`
	Jitter  float64 `json:"jitter,omitempty"`
	Status  string  `json:"status,omitempty"`
}

func (ts *TimeStat) getNTPInfo() {
	// try common locations first, then search PATH, this could cover some
	// contitions when PATH is not correctly set on calling `collector`
	var syncdBinPaths = []string{"/usr/sbin/ntpq", "/usr/bin/ntpq", "ntpq"}
	var syncd string
	var err error
	for _, syncdPath := range syncdBinPaths {
		if syncd, err = exec.LookPath(syncdPath); err == nil {
			// use the first found exec
			break
		}
		ts.Ver = err.Error()
	}
	// when no `ntpq` found, just return
	if syncd == "" {
		return
	}

	cmd := exec.Command(syncd, "-c rv", "127.0.0.1")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	// set default sync status to none
	ts.Sync = "none"

	output := strings.FieldsFunc(out.String(), multi_split)
	for _, kv := range output {
		tmp := strings.Split(strings.TrimSpace(kv), "=")
		switch {
		case tmp[0] == "version":
			ts.Ver = strings.Trim(tmp[1], "\"")
		case tmp[0] == "stratum":
			ts.Stratum, err = strconv.Atoi(tmp[1])
			if err != nil {
				log.Fatal(err)
			}
		case tmp[0] == "offset":
			ts.Offset, err = strconv.ParseFloat(tmp[1], 64)
			if err != nil {
				log.Fatal(err)
			}
		case tmp[0] == "sys_jitter":
			ts.Jitter, err = strconv.ParseFloat(tmp[1], 64)
			if err != nil {
				log.Fatal(err)
			}
		case strings.Contains(tmp[0], "sync"):
			ts.Sync = tmp[0]
		case len(tmp) > 2 && strings.Contains(tmp[1], "status"):
			// sample line of tmp: ["associd", "0 status", "0618 leap_none"]
			ts.Status = strings.Split(tmp[2], " ")[0]
		default:
			continue
		}
	}
}

func multi_split(r rune) bool {
	switch r {
	case ',':
		return true
	case '\n':
		return true
	default:
		return false
	}
}
