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
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/errors"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
)

const (
	maxConcurrent = 10
	blockSize     = 50 * 1024 * 1024
)

type MetaData struct {
	ContentLen  int64
	ETag        string
	AcceptBytes string
	FileName    string
}

type filePart struct {
	from int64
	to   int64
}

type DownloadOptions struct {
	FileAlias string
	FileUUID  string
	ClusterID uint64
	ClientOptions
}

type FileIDList struct {
	Ids []string
}

func ParseURL(opt *DownloadOptions, url string) error {
	findex := strings.LastIndex(url, "=")
	if findex < 0 {
		return errors.New("invalid url")
	}
	opt.FileUUID = url[findex+1:]

	eindex := strings.Index(url, "/diag")
	if eindex < 0 {
		return errors.New("invalid url")
	}
	opt.Endpoint = url[:eindex]

	return nil
}

func Download(ctx context.Context, opt *DownloadOptions) error {
	logger := ctx.Value(logprinter.ContextKeyLogger).(*logprinter.Logger)
	fmt.Println("\nStart to download file...")
	meta, err := fetchMeta(opt)
	if err != nil {
		return err
	}

	concurrent := computeConcurrent(meta.ContentLen)
	eachSize := meta.ContentLen / int64(concurrent)

	jobs := make([]filePart, concurrent)

	for i := range jobs {
		if i == 0 {
			jobs[i].from = 0
		} else {
			jobs[i].from = jobs[i-1].to + 1
		}
		if i < concurrent-1 {
			jobs[i].to = jobs[i].from + eachSize
		} else {
			jobs[i].to = meta.ContentLen - 1
		}
	}

	dir, err := os.Getwd()
	if err != nil {
		logger.Infof("get current directory failed, use the default directory /tmp")
		dir = "/tmp"
	}

	path := filepath.Join(dir, meta.FileName+".tmp")
	tmpFile := new(os.File)
	if exists(path) {
		tmpFile, err = os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return err
		}
	} else {
		tmpFile, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	defer tmpFile.Close()

	var wg sync.WaitGroup
	for _, j := range jobs {
		wg.Add(1)
		go func(job filePart) {
			defer wg.Done()

			err := DownloadFilePart(job, tmpFile, opt)
			if err != nil {
				panic(fmt.Sprintf("download file part failed, err=%v", err))
			}
			logger.Infof("file part is done, from: %d, to: %d", job.from, job.to)
		}(j)
	}
	wg.Wait()

	localFileName := meta.FileName
	if exists(filepath.Join(dir, meta.FileName)) {
		localFileName = fmt.Sprintf("%s-%d", localFileName, time.Now().UnixNano())
	}

	err = os.Rename(filepath.Join(dir, meta.FileName+".tmp"), filepath.Join(dir, localFileName))
	if err != nil {
		return err
	}

	fmt.Printf("the file %s is downloaded successfully!\n", meta.FileName)

	return nil
}

func computeConcurrent(fileSize int64) int {
	if fileSize <= blockSize {
		return 1
	}

	var concurrent int
	if fileSize%blockSize == 0 {
		concurrent = int(fileSize / blockSize)
	} else {
		concurrent = int(fileSize/blockSize) + 1
	}

	return int(math.Min(float64(concurrent), float64(maxConcurrent)))
}

func fetchMeta(opt *DownloadOptions) (*MetaData, error) {
	req, err := http.NewRequest(http.MethodHead, fmt.Sprintf("%s/api/internal/files/%s", opt.Endpoint, opt.FileUUID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return nil, errors.Errorf("download file failed, msg=%s", resp.Status)
	}

	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return nil, errors.Errorf("server unsupport ranges")
	}

	length, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return nil, errors.Errorf("read content length failed, err=%v", err)
	}

	return &MetaData{
		ContentLen:  int64(length),
		ETag:        resp.Header.Get("ETag"),
		AcceptBytes: resp.Header.Get("Accept-Ranges"),
		FileName:    parseFileInfoFrom(resp),
	}, nil
}

func DownloadFilePart(part filePart, f *os.File, opt *DownloadOptions) error {
	b := make([]byte, part.to-part.from)
	_, err := f.ReadAt(b, part.from)
	if err == nil {
		if bytes.Compare(make([]byte, part.to-part.from), b) != 0 {
			return nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/%s", opt.Endpoint, opt.FileUUID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.from, part.to))

	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("server return error code: %v", resp.StatusCode))
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(bs) != int(part.to-part.from+1) {
		return errors.New("download file part failed")
	}

	_, err = f.WriteAt(bs, int64(part.from))
	return err
}

func parseFileInfoFrom(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if contentDisposition != "" {
		_, params, err := mime.ParseMediaType(contentDisposition)
		if err == nil {
			return params["filename"]
		}
	}

	filename := filepath.Base(resp.Request.URL.Path)
	return filename
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func DownloadFilesByAlias(ctx context.Context, opt *DownloadOptions) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/alias/%s", opt.Endpoint, opt.FileAlias), nil)
	if err != nil {
		return err
	}

	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("download files failed, msg=%s", string(data))
	}

	var idResp FileIDList
	err = json.Unmarshal(data, &idResp)
	if err != nil {
		return err
	}

	if len(idResp.Ids) == 0 {
		return errors.New("not found files by alias name")
	}

	for _, id := range idResp.Ids {
		nopt := DownloadOptions{
			ClientOptions: opt.ClientOptions,
			FileUUID:      id,
		}

		if err = Download(ctx, &nopt); err != nil {
			return err
		}
	}

	return nil
}

func DownloadFilesByClusterID(ctx context.Context, opt *DownloadOptions) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/cluster/%d", opt.Endpoint, opt.ClusterID), nil)
	if err != nil {
		return err
	}

	resp, err := requestWithAuth(&opt.ClientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("download files failed, msg=%s", string(data))
	}

	var idResp FileIDList
	err = json.Unmarshal(data, &idResp)
	if err != nil {
		return err
	}

	if len(idResp.Ids) == 0 {
		return errors.New("not found files by alias name")
	}

	for _, id := range idResp.Ids {
		nopt := DownloadOptions{
			ClientOptions: opt.ClientOptions,
			FileUUID:      id,
		}

		if err = Download(ctx, &nopt); err != nil {
			return err
		}
	}

	return nil
}
