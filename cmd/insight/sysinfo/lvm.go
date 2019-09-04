// Copyright © 2016 Zlatko Čalušić
//
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file.

package sysinfo

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// LogicalVolume information.
type LogicalVolume struct {
	LVName string  `json:"lvname,omitempty"`
	VGName string  `json:"vgname,omitempty"`
	LVSize float64 `json:"lvsize,omitempty"`
}

func (si *SysInfo) getLVMInfo() {
	lvs, err := logicalVolumes()
	if err != nil {
		si.LVM = lvs
		return
	}
	si.LVM = lvs
}

func logicalVolumes() (lvs []LogicalVolume, err error) {
	lvsOutput, delimiter, err := getLogicalVolumes()
	if err != nil {
		return []LogicalVolume{}, err
	}
	lvsLines := strings.Split(lvsOutput, "\n")
	for _, lvLine := range lvsLines {
		if len(lvLine) > 0 {
			lv, err := parseLine(lvLine, delimiter)
			if err != nil {
				return lvs, err
			}
			lvs = append(lvs, lv)
		}
	}
	return
}

func parseLine(lvsLine string, delimiter string) (lv LogicalVolume, rr error) {
	var err error
	tokens := strings.Split(strings.Trim(lvsLine, " "), delimiter)
	if len(tokens) != 3 {
		err = errors.New("expected 3 colon items from lvs, perhaps an unsupported operating system")
		return LogicalVolume{}, err
	}
	lv.LVName = tokens[0]
	lv.VGName = tokens[1]

	tokens[2] = strings.Replace(tokens[2], ",", ".", -1)

	lv.LVSize, err = strconv.ParseFloat(tokens[2], 32)
	if err != nil {
		return LogicalVolume{}, err
	}

	return lv, nil
}

func getLogicalVolumes() (output string, delimiter string, err error) {
	delimiter = ":"
	output, err = runCommand("lvs", "--units=m", "--separator=:", "--nosuffix", "--noheadings", "--options=lv_name,vg_name,lv_size")
	return
}

func runCommand(name string, args ...string) (output string, err error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	output = fmt.Sprintf("%s", out)
	if err != nil {
		return
	}
	return
}
