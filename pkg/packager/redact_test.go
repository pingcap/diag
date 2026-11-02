// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packager

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pingcap/diag/pkg/crypto"
	"github.com/pingcap/diag/pkg/redact"
	"github.com/stretchr/testify/require"
)

func TestRedactedPackageEncryptionRoundTrip(t *testing.T) {
	t.Run("copy-question-marks", func(t *testing.T) { testRedactedPackageRoundTrip(t, false) })
	t.Run("in-place-partial", func(t *testing.T) { testRedactedPackageRoundTrip(t, true) })
}

func testRedactedPackageRoundTrip(t *testing.T, inPlace bool) {
	root := t.TempDir()
	input := filepath.Join(root, "input")
	require.NoError(t, os.MkdirAll(filepath.Join(input, "10.1.1.1"), 0700))
	cluster := `{"cluster_id":"123456789","cluster_type":"tidb-cluster","cluster_name":"test","topology":{"host":"10.1.1.1","log_files":["10.1.1.1/tidb.log"]}}`
	require.NoError(t, os.WriteFile(filepath.Join(input, "cluster.json"), []byte(cluster), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(input, "10.1.1.1/tidb.log"), []byte("host=10.1.1.1:4000\n"), 0600))
	r, err := redact.Discover(input)
	require.NoError(t, err)
	var redacted string
	expectedIP, expectedLog := "198.18.0.1", "host=?:4000\n"
	if inPlace {
		require.NoError(t, r.KeepIPv4Octets(1))
		redacted, _, err = r.Rewrite(input)
		expectedIP, expectedLog = "10.0.0.1", "host=10.0.0.1:4000\n"
	} else {
		require.NoError(t, r.MaskLogIPv4())
		redacted, _, err = r.Copy(input, "")
	}
	require.NoError(t, err)
	// Use a fresh test-only key, never a production certificate/private key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	output := filepath.Join(root, "result.diag")
	_, err = PackageCollectedData(&PackageOptions{InputDir: redacted, OutputFile: output, Cert: string(cert)}, true)
	require.NoError(t, err)
	packed, err := os.ReadFile(output)
	require.NoError(t, err)
	meta, format, _, offset, err := ParserD1agHeader(bytes.NewReader(packed))
	require.NoError(t, err)
	require.Equal(t, "diag", format)
	metadata, err := crypto.NewDecryptor(key, bytes.NewReader(meta))
	require.NoError(t, err)
	body, err := io.ReadAll(metadata)
	require.NoError(t, err)
	require.Contains(t, string(body), `"cluster_id":"123456789"`)
	decrypted, err := crypto.NewDecryptor(key, bytes.NewReader(packed[offset:]))
	require.NoError(t, err)
	decoder, err := zstd.NewReader(decrypted)
	require.NoError(t, err)
	defer decoder.Close()
	tr := tar.NewReader(decoder)
	files := make(map[string]string)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.NotContains(t, header.Name, "10.1.1.1")
		if header.FileInfo().IsDir() {
			continue
		}
		body, err := io.ReadAll(tr)
		require.NoError(t, err)
		require.NotContains(t, string(body), "10.1.1.1")
		files[header.Name] = string(body)
	}
	require.Equal(t, expectedLog, files[expectedIP+"/tidb.log"])
	require.Contains(t, files["cluster.json"], `"log_files":["`+expectedIP+`/tidb.log"]`)
}
