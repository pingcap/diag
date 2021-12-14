// Copyright 2021 PingCAP, Inc.
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

package packager

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type Histroy struct {
	file string
	list []string
}

func LoadHistroy() (*Histroy, error) {
	dir := os.Getenv("TIUP_COMPONENT_DATA_DIR")
	if dir == "" {
		dir = path.Join(os.Getenv("HOME"), ".clinic")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	file := path.Join(dir, "history.txt")
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	list := strings.Split(string(data), "\n")
	return &Histroy{file, list}, nil
}

func (h *Histroy) Push(url string) {
	h.list = append([]string{url}, h.list...)
}

func (h *Histroy) Store() error {
	list := h.list
	if len(list) > 10 {
		list = list[:10]
	}
	return os.WriteFile(h.file, []byte(strings.Join(list, "\n")), 0664)
}

func (h *Histroy) PrintList() {
	for _, url := range h.list {
		fmt.Println(url)
	}
}
