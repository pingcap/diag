// Copyright 2022 PingCAP, Inc.
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

package utils

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// DirSize returns the total file size of a dir
func DirSize(dir string) (int64, error) {
	var totalSize int64
	if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return err
	}); err != nil {
		return totalSize, err
	}
	return totalSize, nil
}

// TrimLeftSpace trim all left space
func TrimLeftSpace(s string) string {
	return strings.TrimLeft(s, "\t\n\v\f\r ")
}

// AddHeaders parse headers like "a: b" and add them to exist header
func AddHeaders(exist http.Header, addons []string) {
	for _, line := range addons {
		index := strings.IndexRune(line, ':')
		exist.Add(line[:index], TrimLeftSpace(line[index+1:]))
	}
}
