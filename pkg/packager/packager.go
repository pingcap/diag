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
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/crypto"
	"github.com/pingcap/tiup/pkg/tui"
	tiuputils "github.com/pingcap/tiup/pkg/utils"
)

type PackageOptions struct {
	InputDir   string // source directory of collected data
	OutputFile string // target file to store packaged data
	CertPath   string // crt file to encrypt data
}

func PackageCollectedData(pOpt *PackageOptions, skipConfirm bool) (string, error) {

	if tiuputils.IsNotExist(filepath.Dir(pOpt.OutputFile)) {
		os.MkdirAll(filepath.Dir(pOpt.OutputFile), 0755)
	}

	input, err := selectInputDir(pOpt.InputDir, skipConfirm)
	if err != nil {
		return "", err
	}

	output, err := selectOutputFile(input, pOpt.OutputFile)
	if err != nil {
		return "", err
	}

	certPath, err := selectCertFile(pOpt.CertPath)
	if err != nil {
		return "", err
	}

	certString, err := os.ReadFile(certPath)
	if err != nil {
		return "", err
	}
	block, _ := pem.Decode(certString)
	cert, _ := x509.ParseCertificate(block.Bytes)
	publicKey := cert.PublicKey.(*rsa.PublicKey)

	fileW, _ := os.Create(output)
	defer fileW.Close()
	encryptW, _ := crypto.NewEncryptWriter(publicKey, fileW)
	compressW, _ := zstd.NewWriter(encryptW)
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
		defer fd.Close()
		io.Copy(tarW, fd)
		return nil
	})

	return output, nil
}

func selectInputDir(dir string, skipConfirm bool) (string, error) {
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
			return "", fmt.Errorf("input directory not specified and can not be auto detected")
		}
		fmt.Printf("found possible input directory: %s\n", dir)

		if skipConfirm {
			return dir, nil
		}

		err = tui.PromptForConfirmOrAbortError("Do you want to use it? [y/N]: ")
		if err != nil {
			return dir, err
		}
	}

	_, err := os.Stat(filepath.Join(dir, "cluster.json"))
	if err != nil {
		return "", fmt.Errorf("%s is not a diag collected data directory", dir)
	}
	return filepath.Abs(dir)
}

func selectOutputFile(input, output string) (string, error) {
	if output == "" {
		output = filepath.Base(input) + ".diag"
	}
	_, err := os.Stat(output)
	if err == nil {
		return output, fmt.Errorf("%s already exists", output)
	}
	return filepath.Abs(output)
}

func selectCertFile(path string) (string, error) {
	// choose latest diag directory if not specify
	if path == "" {
		path = filepath.Join(filepath.Dir(os.Args[0]), "pingcap.crt")
	}
	_, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot find cert for encryption: %w", err)
	}
	return filepath.Abs(path)
}
