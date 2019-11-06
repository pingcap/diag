/// This directory checks the transparent huge page for the instance application.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	log "github.com/sirupsen/logrus"
)

type Memory struct {
	// Type of Transparent Huge page
	ThpType string `json:"thp_type"`
}

//
func (m* Memory) GetMemory() {
	thp, err := collectThp()
	if err != nil {
		log.Fatal(err)
	}
	// it will not be nil if err is not nil
	m.ThpType = *thp
}

type TransparentHugePage string

var thpMap map[string]TransparentHugePage

const (
	Always  TransparentHugePage = "always"
	MAdvise TransparentHugePage = "madvise"
	Never   TransparentHugePage = "never"
)

func init() {
	thpMap = make(map[string]TransparentHugePage)

	thpMap["always"] = Always
	thpMap["madvise"] = MAdvise
	thpMap["never"] = Never
}

func catchString(s string) (*TransparentHugePage, error) {
	re := regexp.MustCompile(`\[.+\]`)
	regResp := re.FindString(s)
	if len(regResp) == 0 {
		return nil, errors.New(fmt.Sprintf("cat failed, %s", s))
	}

	regResp = regResp[1 : len(regResp)-1]

	// must larger than 2
	if page, ok := thpMap[regResp]; !ok {
		return nil, errors.New(fmt.Sprintf("status %s not exists", page))
	} else {
		return &page, nil
	}
}

// TODO: this part of logic is paste from config/config, please find method to modify this.
func collectThp() (*string, error) {

	// Note: this method can only be used in
	cmd := exec.Command(
		fmt.Sprintf("cat /sys/kernel/mm/redhat_transparent_huge"),
	)
	// parse buffer from thp
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = os.Stderr

	log.Info(cmd.Args)
	if err := cmd.Run(); err != nil {
		log.Error("collect config file:", err)
		return nil, err
	}

	resp, err := catchString(out.String())
	if err != nil {
		return nil, err
	}
	strResp := string(*resp)
	return &strResp, nil
}
