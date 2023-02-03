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
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/version"
	"github.com/pingcap/errors"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
)

type preCreateResponse struct {
	Partseq    int   `json:"sequence"`
	BlockBytes int64 `json:"blockbytes"`
}

type FlushResponse struct {
	ResultURL string `json:"result"`
}

type UploadOptions struct {
	FilePath    string
	Alias       string
	Issue       string
	Concurrency int
	Rebuild     bool
	Cert        string
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
				Cert:       opt.Cert,
				Rebuild:    opt.Rebuild,
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
	req.Header.Add("Authorization", "Bearer "+opt.Token)

	appendClinicHeader(req)

	resp, err := opt.Client.Do(req)
	if err != nil {
		return nil, errors.Errorf("preupload file failed, error is %s", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("preupload file failed, status code is %d,msg=\n%v", resp.StatusCode, string(data))
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
		return "", fmt.Errorf("upload failed: %s", err)
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
	q.Add("issue", opt.Issue)
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+opt.Token)

	appendClinicHeader(req)
	resp, err := opt.Client.Do(req)
	if err != nil {
		return "", errors.Errorf("flush file failed, error is %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("flush file failed, status code is %d, msg=\n%v", resp.StatusCode, string(body))
	}

	var result FlushResponse
	dc := json.NewDecoder(resp.Body)
	err = dc.Decode(&result)
	if err != nil {
		return "", err
	}
	logger.Infof("Completed!")
	logger.Infof("Download URL: %s\n", result.ResultURL)

	if his, err := LoadHistroy(); err == nil {
		his.Push(result.ResultURL)
		his.Store()
	}

	return result.ResultURL, nil
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
	req.Header.Add("Authorization", "Bearer "+opt.Token)

	appendClinicHeader(req)
	resp, err := opt.Client.Do(req)
	if err != nil {
		return errors.Errorf("upload part failed, error is %s", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("upload part failed, status code is %d, msg=\n%v", resp.StatusCode, string(body))
	}

	return nil
}
