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
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/tiup/pkg/tui"
)

type PackageOptions struct {
	InputDir   string // source directory of collected data
	OutputFile string // target file to store packaged data
	Compress   string
}

func PackageCollectedData(pOpt *PackageOptions) error {
	input, err := selectInputDir(pOpt.InputDir)
	if err != nil {
		return err
	}

	suffix, _ := selectSuffix(pOpt.Compress)
	output, err := selectOutputFile(input, pOpt.OutputFile, suffix)
	if err != nil {
		return err
	}

	fileW, _ := os.Create(output)
	defer fileW.Close()
	compressW := newWriterByCompress(fileW, pOpt.Compress)
	defer compressW.Close()
	tarW := tar.NewWriter(compressW)
	defer tarW.Close()

	filepath.Walk(input, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, _ := tar.FileInfoHeader(info, "")
		header.Name, _ = filepath.Rel(input, path)
		//skip "."
		if header.Name == "." {
			return nil
		}

		err = tarW.WriteHeader(header)
		if err != nil {
			return err
		}
		fd, err := os.Open(path)
		if err != nil {
			return err
		}
		io.Copy(tarW, fd)
		return nil
	})

	return nil
}

func selectInputDir(dir string) (string, error) {
	// choose latest diag directory if not specify
	if dir == "" {
		fileInfos, err := os.ReadDir(".")
		if err != nil {
			return dir, err
		}
		for _, fileInfo := range fileInfos {
			info, err := os.Stat(filepath.Join(".", fileInfo.Name()))
			if err != nil {
				return dir, err
			}
			if info.IsDir() && strings.HasPrefix(fileInfo.Name(), "diag-") {
				dir = fileInfo.Name()
			}
		}
		if dir == "" {
			return "", fmt.Errorf("cannot find input directory")
		}
		fmt.Printf("found possible input directory: %s\n", dir)
		err = tui.PromptForConfirmOrAbortError("Do you want to use it? [y/N]: ")
		if err != nil {
			return dir, err
		}
	}
	// TBD: check cluster.json
	_, err := os.Stat(filepath.Join(dir, "cluster-name.txt"))
	if err != nil {
		return "", fmt.Errorf("%s is not a diag collected data directory", dir)
	}
	return filepath.Abs(dir)
}

func newWriterByCompress(w io.Writer, compress string) io.WriteCloser {
	switch compress {
	case "", "gzip":
		return gzip.NewWriter(w)
	case "zstd":
		zw, _ := zstd.NewWriter(w)
		return zw
	default:
		return gzip.NewWriter(w)
	}
}

func selectSuffix(compress string) (string, error) {
	var err error
	suffix := map[string]string{
		"":     ".tar.gz",
		"gzip": ".tar.gz",
		"zstd": ".tar.zst",
	}
	if suffix[compress] == "" {
		err = fmt.Errorf("%s is not supported algorithm", compress)
	}

	return suffix[compress], err
}

func selectOutputFile(input, output, outputSuffix string) (string, error) {
	if output == "" {
		output = input
	}
	output = filepath.Base(output) + outputSuffix
	_, err := os.Stat(output)
	if err == nil {
		return output, fmt.Errorf("%s already exists", output)
	}
	return filepath.Abs(output)
}
