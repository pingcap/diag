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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/version"
	"github.com/pingcap/errors"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
)

type preCreateResponse struct {
	Partseq    int
	BlockBytes int64
}

type UploadOptions struct {
	FilePath    string
	Alias       string
	Issue       string
	Concurrency int
	ClientOptions
}

type ClientOptions struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

func Upload(ctx context.Context, opt *UploadOptions, skipConfirm bool) (string, error) {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	fileStat, err := os.Stat(opt.FilePath)
	if err != nil {
		return "", err
	}
	if fileStat.IsDir() {
		dataDir := opt.FilePath
		opt.FilePath, err = selectOutputFile(dataDir, "")
		// err means it is already packaged
		if err == nil {
			logger.Infof("packaging collected data...")
			_, err = PackageCollectedData(&PackageOptions{
				InputDir:   dataDir,
				OutputFile: opt.FilePath,
				CertPath:   "", // use default cert in install path
			}, skipConfirm)
			if err != nil {
				return "", err
			}
		}

		fileStat, err = os.Stat(opt.FilePath)
		if err != nil {
			return "", err
		}
	}

	uuid := fmt.Sprintf("%s-%s-%s",
		fnvHash(logger, fileStat.Name()),
		fnvHash(logger, fmt.Sprintf("%d", fileStat.Size())),
		fnvHash(logger, fileStat.ModTime().Format(time.RFC3339)),
	)
	if opt.Alias != "" {
		uuid = fmt.Sprintf("%s-%s", uuid, fnvHash32(logger, opt.Alias))
	}

	f, err := os.Open(opt.FilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	meta, encryption, compress, offset, err := ParserD1agHeader(f)
	if err != nil {
		return "", err
	}

	presp, err := preCreate(uuid, fileStat.Size()-int64(offset), fileStat.Name(), meta, encryption, compress, opt)
	if err != nil {
		return "", err
	}

	return UploadFile(
		logger,
		opt.Concurrency,
		presp,
		fileStat.Size(),
		func() (string, error) {
			return UploadComplete(logger, uuid, opt)
		},
		func() (io.ReadSeekCloser, error) {
			rec, err := os.Open(opt.FilePath)
			if err != nil {
				return nil, err
			}
			rec.Seek(int64(offset), 0)
			return rec, nil
		},
		func(serial, size int64, r io.Reader) error {
			return uploadMultipartFile(uuid, serial, size, r, opt)
		},
	)
}

func preCreate(uuid string, fileLen int64, originalName string, meta []byte, encryption, compress string, opt *UploadOptions) (*preCreateResponse, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/clinic/api/v1/diag/precreate", opt.Endpoint), bytes.NewBuffer(meta))
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("uuid", uuid)
	q.Add("length", fmt.Sprintf("%d", fileLen))
	q.Add("alias", opt.Alias)
	q.Add("filename", originalName)
	q.Add("encryption", encryption)
	q.Add("compression", compress)
	req.URL.RawQuery = q.Encode()

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("preupload file failed, msg=%s", string(data))
	}

	var presp preCreateResponse
	err = json.Unmarshal(data, &presp)
	if err != nil {
		return nil, err
	}

	return &presp, nil
}

func computeTotalBlock(fileSize int64, blockSize int64) int {
	if fileSize%blockSize == 0 {
		return int(fileSize / blockSize)
	}

	return int(fileSize/blockSize) + 1
}

type OpenFunc func() (io.ReadSeekCloser, error)

type UploadPart func(int64, int64, io.Reader) error

type FlushUploadFile func() (string, error)

func UploadFile(
	logger *logprinter.Logger,
	concurrency int,
	presp *preCreateResponse,
	fileSize int64,
	flush FlushUploadFile,
	open OpenFunc,
	uploadPart UploadPart,
) (string, error) {
	totalBlock := computeTotalBlock(fileSize, presp.BlockBytes)
	if totalBlock <= presp.Partseq {
		return flush()
	}
	errChan := make(chan error, totalBlock)

	go concurrentUploadFile(logger, concurrency, presp, totalBlock, fileSize, open, uploadPart, errChan)

	// catch errors
	// errChan is not closed or the error can be obtained, exit
	if err, ok := <-errChan; ok {
		logger.Errorf("cat: upload file failed: %s\n", err)
		return "", fmt.Errorf("upload file failed: %s", err)
	}

	return flush()
}

// concurrentUploadFile  concurrent execute the function that actually uploads the file
func concurrentUploadFile(
	logger *logprinter.Logger,
	concurrency int,
	presp *preCreateResponse,
	totalBlock int,
	fileSize int64,
	open OpenFunc,
	uploadPart UploadPart,
	errChan chan error,
) {

	waitGroup := sync.WaitGroup{}
	if concurrency < 1 {
		concurrency = 1
	}
	for c := 0; c < concurrency; c++ {
		i := int64(presp.Partseq) + int64(c)
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for ; i < int64(totalBlock); i = i + int64(concurrency) {
				eachSize := presp.BlockBytes
				if i == int64(totalBlock)-1 {
					eachSize = fileSize - i*presp.BlockBytes
				}

				f, err := open()
				if err != nil {
					errChan <- err
					return
				}
				defer f.Close()

				f.Seek(i*presp.BlockBytes, 1)
				partR := io.LimitReader(f, eachSize)

				if logger.GetDisplayMode() == logprinter.DisplayModeDefault {
					fmt.Printf(">")
				}

				if err = uploadPart(i+1, eachSize, partR); err != nil {
					errChan <- err
					return
				}

				if logger.GetDisplayMode() == logprinter.DisplayModeDefault {
					fmt.Printf(">")
				}
			}
		}()
	}

	// all goroutines are executed
	waitGroup.Wait()
	close(errChan)
}

func appendClinicHeader(req *http.Request) {
	req.Header.Add("x-clinic-client", "upload")
	req.Header.Add("x-diag-version", version.ReleaseVersion)
}

func UploadComplete(logger *logprinter.Logger, fileUUID string, opt *UploadOptions) (string, error) {
	fmt.Println("<>>>>>>>>>")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/clinic/api/v1/diag/flush", opt.Endpoint), nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	q.Add("uuid", fileUUID)
	q.Add("Issue", opt.Issue)
	req.URL.RawQuery = q.Encode()

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("upload file failed, msg=%v", string(body))
	}

	url := string(body)
	logger.Infof("Completed!")
	logger.Infof("Download URL: %s\n", url)

	if his, err := LoadHistroy(); err == nil {
		his.Push(url)
		his.Store()
	}

	return url, nil
}

func fnvHash(logger *logprinter.Logger, raw string) string {
	hash := fnv.New64()
	if _, err := hash.Write([]byte(raw)); err != nil {
		// impossible path
		logger.Errorf("failed to write fnv hash: %s", err)
	}
	return fmt.Sprintf("%x", hash.Sum64())
}

func fnvHash32(logger *logprinter.Logger, raw string) string {
	hash := fnv.New32()
	if _, err := hash.Write([]byte(raw)); err != nil {
		// impossible path
		logger.Errorf("failed to write fnv hash: %s", err)
	}
	return fmt.Sprintf("%x", hash.Sum32())
}

func uploadMultipartFile(fileUUID string, serialNum, size int64, r io.Reader, opt *UploadOptions) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/clinic/api/v1/diag/upload", opt.Endpoint), r)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("uuid", fileUUID)
	q.Add("sequence", fmt.Sprintf("%d", serialNum))
	q.Add("length", fmt.Sprintf("%d", size))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Content-Type", "application/octet-stream")

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("upload file failed, msg=%v", string(body))
	}

	return nil
}

func InitClient(Endpoint string) *http.Client {
	if strings.HasPrefix(strings.ToLower(Endpoint), "https://") {
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM([]byte(privateCA))
		if !ok {
			panic("failed to parse root certificate")
		}
		ok = roots.AppendCertsFromPEM([]byte(publicCA))
		if !ok {
			panic("failed to parse root certificate")
		}
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12},
			},
		}

	}

	return http.DefaultClient
}

var privateCA = `-----BEGIN CERTIFICATE-----
MIIDizCCAnOgAwIBAgIRAJK3FPu59GQ+fCx9skKPQwUwDQYJKoZIhvcNAQELBQAw
XzELMAkGA1UEBhMCQ04xEDAOBgNVBAoMB3BpbmdjYXAxDTALBgNVBAsMBHRpZGIx
DjAMBgNVBAgMBUNoaW5hMQ0wCwYDVQQDDARDQSAxMRAwDgYDVQQHDAdjaGVuZ2R1
MB4XDTIxMDIyNDA0MDI0MFoXDTQxMDIyNDA1MDI0MFowXzELMAkGA1UEBhMCQ04x
EDAOBgNVBAoMB3BpbmdjYXAxDTALBgNVBAsMBHRpZGIxDjAMBgNVBAgMBUNoaW5h
MQ0wCwYDVQQDDARDQSAxMRAwDgYDVQQHDAdjaGVuZ2R1MIIBIjANBgkqhkiG9w0B
AQEFAAOCAQ8AMIIBCgKCAQEAlNsKtdOUXM9M08PWYk3cRHfa+DeQYClz4hiTZbun
1vgcglKreaQ2Q6DVG3W64x+JlQZqfB4Rzj0xO+6sTrO0b5/Ou6uxx9C1ywvK6SjO
23wjkZPp0Rd84vyzcF6uanq9e3SzjrahJaWwTMHaYXqDpVs6w0K4aYo2T1TqQoMY
gdXu/7lcerYAjdBhaLSY68TuS/sW4CSanpHcH8jzRwm/PLuvO+HJ6p+v1DMAZxl8
FF1K+ivUSMKNkwHw6j5vF5hKCyCJYCqn49NYAV+K914WN8fI2KaLpD1zwr7FoPLQ
bGCjZWFGXeJ7R/4+uK3IFighr/7NKuBf78Szb3Ms3ZtKMwIDAQABo0IwQDAPBgNV
HRMBAf8EBTADAQH/MB0GA1UdDgQWBBQT/cQTJebkFLih8qhNHXleHIxF9jAOBgNV
HQ8BAf8EBAMCAYYwDQYJKoZIhvcNAQELBQADggEBAFR4Ic7I/pMdXiomDW5a5HIi
dk19KDIBIvvF3onLLy5aN6P11j2z57yLblXpTLoGowsc0FqtYoxczv768cyG0JJI
MBmSW+krcqHIg6GXMhMzekNuwL/ae6fLFefyGgAwo5GhD4t02jFDA0mspUNoI0gX
38DYxUABs5EPOWdOLfLHXKxYvLx1Qs2sNjDKKppPgs5Jw8g2MiKYmpDvXHtA6N3B
6JjQ5AbHOz05Yu2/NhwmMbmSNXH8hUJJYg9zhyGd9YiXF+6r5fj4zaOYNY7cP0UN
bzt89DIiAIVY/SxoonK/u5myE3h8ChdYMbdeUCBBuIWFpzK5EfbFjmV90dMgxDw=
-----END CERTIFICATE-----`

var publicCA = `-----BEGIN CERTIFICATE-----
MIIGEzCCA/ugAwIBAgIQfVtRJrR2uhHbdBYLvFMNpzANBgkqhkiG9w0BAQwFADCB
iDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCk5ldyBKZXJzZXkxFDASBgNVBAcTC0pl
cnNleSBDaXR5MR4wHAYDVQQKExVUaGUgVVNFUlRSVVNUIE5ldHdvcmsxLjAsBgNV
BAMTJVVTRVJUcnVzdCBSU0EgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkwHhcNMTgx
MTAyMDAwMDAwWhcNMzAxMjMxMjM1OTU5WjCBjzELMAkGA1UEBhMCR0IxGzAZBgNV
BAgTEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4GA1UEBxMHU2FsZm9yZDEYMBYGA1UE
ChMPU2VjdGlnbyBMaW1pdGVkMTcwNQYDVQQDEy5TZWN0aWdvIFJTQSBEb21haW4g
VmFsaWRhdGlvbiBTZWN1cmUgU2VydmVyIENBMIIBIjANBgkqhkiG9w0BAQEFAAOC
AQ8AMIIBCgKCAQEA1nMz1tc8INAA0hdFuNY+B6I/x0HuMjDJsGz99J/LEpgPLT+N
TQEMgg8Xf2Iu6bhIefsWg06t1zIlk7cHv7lQP6lMw0Aq6Tn/2YHKHxYyQdqAJrkj
eocgHuP/IJo8lURvh3UGkEC0MpMWCRAIIz7S3YcPb11RFGoKacVPAXJpz9OTTG0E
oKMbgn6xmrntxZ7FN3ifmgg0+1YuWMQJDgZkW7w33PGfKGioVrCSo1yfu4iYCBsk
Haswha6vsC6eep3BwEIc4gLw6uBK0u+QDrTBQBbwb4VCSmT3pDCg/r8uoydajotY
uK3DGReEY+1vVv2Dy2A0xHS+5p3b4eTlygxfFQIDAQABo4IBbjCCAWowHwYDVR0j
BBgwFoAUU3m/WqorSs9UgOHYm8Cd8rIDZsswHQYDVR0OBBYEFI2MXsRUrYrhd+mb
+ZsF4bgBjWHhMA4GA1UdDwEB/wQEAwIBhjASBgNVHRMBAf8ECDAGAQH/AgEAMB0G
A1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAbBgNVHSAEFDASMAYGBFUdIAAw
CAYGZ4EMAQIBMFAGA1UdHwRJMEcwRaBDoEGGP2h0dHA6Ly9jcmwudXNlcnRydXN0
LmNvbS9VU0VSVHJ1c3RSU0FDZXJ0aWZpY2F0aW9uQXV0aG9yaXR5LmNybDB2Bggr
BgEFBQcBAQRqMGgwPwYIKwYBBQUHMAKGM2h0dHA6Ly9jcnQudXNlcnRydXN0LmNv
bS9VU0VSVHJ1c3RSU0FBZGRUcnVzdENBLmNydDAlBggrBgEFBQcwAYYZaHR0cDov
L29jc3AudXNlcnRydXN0LmNvbTANBgkqhkiG9w0BAQwFAAOCAgEAMr9hvQ5Iw0/H
ukdN+Jx4GQHcEx2Ab/zDcLRSmjEzmldS+zGea6TvVKqJjUAXaPgREHzSyrHxVYbH
7rM2kYb2OVG/Rr8PoLq0935JxCo2F57kaDl6r5ROVm+yezu/Coa9zcV3HAO4OLGi
H19+24rcRki2aArPsrW04jTkZ6k4Zgle0rj8nSg6F0AnwnJOKf0hPHzPE/uWLMUx
RP0T7dWbqWlod3zu4f+k+TY4CFM5ooQ0nBnzvg6s1SQ36yOoeNDT5++SR2RiOSLv
xvcRviKFxmZEJCaOEDKNyJOuB56DPi/Z+fVGjmO+wea03KbNIaiGCpXZLoUmGv38
sbZXQm2V0TP2ORQGgkE49Y9Y3IBbpNV9lXj9p5v//cWoaasm56ekBYdbqbe4oyAL
l6lFhd2zi+WJN44pDfwGF/Y4QA5C5BIG+3vzxhFoYt/jmPQT2BVPi7Fp2RBgvGQq
6jG35LWjOhSbJuMLe/0CjraZwTiXWTb2qHSihrZe68Zk6s+go/lunrotEbaGmAhY
LcmsJWTyXnW0OMGuf1pGg+pRyrbxmRE1a6Vqe8YAsOf4vmSyrcjC8azjUeqkk+B5
yOGBQMkKW+ESPMFgKuOXwIlCypTPRpgSabuY0MLTDXJLR27lk8QyKGOHQ+SwMj4K
00u/I5sUKUErmgQfky3xxzlIPK1aEn8=
-----END CERTIFICATE-----`

func requestWithAuth(opt *ClientOptions, req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+opt.Token)
	resp, err := opt.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("401 Unauthorized")
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("403 Clinic server forbid to upload, please check your permission")
	}

	if resp.StatusCode == http.StatusBadRequest {
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New("400 Bad Request")
		}

		errmsg := resp.Status
		if string(data) != "" {
			errmsg = fmt.Sprintf("%s. %s", errmsg, string(data))
		}
		return nil, errors.Errorf("400 Bad Request, msg=%s", errmsg)
	}

	if resp.StatusCode == http.StatusProcessing {
		return nil, errors.New("102 the resource is processing, please again later")
	}

	return resp, nil
}
