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
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
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

// Copy copies a file or directory from src to dst
func Copy(src, dst string) error {
	// check if src is a directory
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		// use copy.Copy to copy a directory
		return copy.Copy(src, dst)
	}

	// for regular files
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = out.Close()
	if err != nil {
		return err
	}

	err = os.Chmod(dst, fi.Mode())
	if err != nil {
		return err
	}

	// Make sure the created dst's modify time is newer (at least equal) than src
	// this is used to workaround github action virtual filesystem
	ofi, err := os.Stat(dst)
	if err != nil {
		return err
	}
	if fi.ModTime().After(ofi.ModTime()) {
		return os.Chtimes(dst, fi.ModTime(), fi.ModTime())
	}
	return nil
}
