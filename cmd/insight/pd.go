// tidb-insight project pd.go
package main

import (
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/process"
)

// PDMeta is the metadata struct of a PD server
type PDMeta struct {
 	MetaBase
}

func getPDVersion(proc *process.Process) PDMeta {
	var pdVer PDMeta
	pdVer.Pid = proc.Pid
	file, err := proc.Exe()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(file, "-V")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	output := strings.Split(out.String(), "\n")
	for _, line := range output {
		info := strings.Split(line, ":")
		if len(info) <= 1 {
			continue
		}
		switch info[0] {
		case "Release Version":
			pdVer.ReleaseVer = strings.TrimSpace(info[1])
		case "Git Commit Hash":
			pdVer.GitCommit = strings.TrimSpace(info[1])
		case "Git Branch":
			pdVer.GitBranch = strings.TrimSpace(info[1])
		case "UTC Build Time":
			pdVer.BuildTime = strings.TrimSpace(strings.Join(info[1:], ":"))
		default:
			continue
		}
	}
	pdVer.ParseLimits(proc)

	return pdVer
}

func getPDVersionByName() []PDMeta {
	var pdMeta = make([]PDMeta, 0)
	procList, err := getProcessesByName("pd-server")
	if err != nil {
		log.Fatal(err)
	}
	if len(procList) < 1 {
		return pdMeta
	}

	for _, proc := range procList {
		pdMeta = append(pdMeta, getPDVersion(proc))
	}
	return pdMeta
}

func getPDVersionByPortList(portList []string) []PDMeta {
	pdMeta := make([]PDMeta, 0)
	for _, portStr := range portList {
		portNum, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatal(err)
		}
		proc, err := getProcessByPort(portNum)
		if err != nil {
			log.Fatal(err)
		}
		if proc == nil {
			continue
		}
		procName, _ := proc.Name()
		if !strings.Contains(procName, "pd-server") {
			continue
		}
		pdMeta = append(pdMeta, getPDVersion(proc))
	}
	return pdMeta
}
