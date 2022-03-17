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
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/crypto"
	"github.com/pingcap/tiup/pkg/tui"
)

type PackageOptions struct {
	InputDir   string // source directory of collected data
	OutputFile string // target file to store packaged data
	CertPath   string // crt file to encrypt data
	Compress   byte
	Meta       map[string]interface{}
}

const (
	TypeTar        = 0
	TypeTarGZ      = 01
	TypeTarZST     = 02
	TypeRaw        = 020
	TypeEncryption = 030
)

// meta not compress
func GenerateD1agHeader(meta map[string]interface{}, format byte, certPath string) ([]byte, error) {
	header := []byte("D1ag")
	packageType := format

	var w io.Writer
	metaBuf := new(bytes.Buffer)

	if certPath == "" {
		packageType |= TypeRaw
		w = metaBuf
	} else {
		// encryption meta information
		packageType |= TypeEncryption
		certString, err := os.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(certString)
		cert, _ := x509.ParseCertificate(block.Bytes)
		publicKey := cert.PublicKey.(*rsa.PublicKey)
		w, _ = crypto.NewEncryptWriter(publicKey, metaBuf)
	}

	j, err := json.Marshal(meta)
	if err != nil {
		return nil, err
	}
	w.Write(j)

	if metaBuf.Len() > 0777 {
		return nil, fmt.Errorf("the meta is too big")
	}
	header = append(header, packageType, byte(metaBuf.Len()>>16), byte(metaBuf.Len()>>8), byte(metaBuf.Len()))
	header = append(header, metaBuf.Bytes()...)
	return header, nil
}

func ParserD1agHeader(r io.Reader) (meta []byte, encryption, compress string, offset int, err error) {
	buf := make([]byte, 8)
	_, err = r.Read(buf)
	if err != nil {
		return nil, "", "", 0, err
	}

	if string(buf[0:4]) != "D1ag" {
		// TBD: forbidden upload non-d1ag file
		return nil, "v1", "tar.zst", 0, nil
	}

	// byte 3~5
	switch buf[4] & 070 {
	case TypeRaw:
		encryption = "none"
	case TypeEncryption:
		encryption = "v1"
	default:
		return nil, "", "", 0, fmt.Errorf("unknown type: %x", buf[4])
	}

	// byte 6~8
	switch buf[4] & 007 {
	case TypeTar:
		compress = "tar"
	case TypeTarGZ:
		compress = "tar.gz"
	case TypeTarZST:
		compress = "tar.zst"
	default:
		return nil, "", "", 0, fmt.Errorf("unknown type: %x", buf[4])
	}

	metaLen := int(buf[5])<<16 + int(buf[6])<<8 + int(buf[7])
	meta = make([]byte, metaLen)
	_, _ = r.Read(meta)
	return meta, encryption, compress, metaLen + 8, nil
}

func PackageCollectedData(pOpt *PackageOptions, skipConfirm bool) (string, error) {
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

	// read cluster name and id
	body, err := os.ReadFile(filepath.Join(input, "cluster.json"))
	if err != nil {
		return "", err
	}
	clusterJSON := make(map[string]interface{})
	d := json.NewDecoder(bytes.NewBuffer(body))
	d.UseNumber()
	err = d.Decode(&clusterJSON)
	if err != nil {
		return "", err
	}

	meta := make(map[string]interface{})
	meta["cluster_id"], meta["cluster_type"], err = validateClusterID(clusterJSON)
	if err != nil {
		return "", err
	}
	meta["begin_time"], meta["end_time"] = clusterJSON["begin_time"], clusterJSON["end_time"]
	header, _ := GenerateD1agHeader(meta, TypeTarZST, certPath)
	fileW.Write(header)

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

func validateClusterID(clusterJSON map[string]interface{}) (clusterID uint64, clusterType string, err error) {
	IDstr := clusterJSON["cluster_id"].(json.Number).String()
	clusterID, err = strconv.ParseUint(IDstr, 10, 64)
	if err != nil {
		return 0, "", err
	}
	clusterType, ok := clusterJSON["cluster_type"].(string)
	if !ok {
		return 0, "", fmt.Errorf("cluster_type must exist in cluster.json")
	}
	if clusterType == "tidb-cluster" && clusterID == 0 {
		return 0, "", fmt.Errorf("cluster_id should not be 0 for tidb cluster, please check if PD is up when collect data")
	}
	return clusterID, clusterType, nil
}
