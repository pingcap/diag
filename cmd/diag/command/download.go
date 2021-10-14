package command

import (
	"bytes"
	"encoding/json"
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

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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

type downloadOptions struct {
	fileAlias string
	fileUUID  string
	clusterID uint64
	clientOptions
}

type FileIDList struct {
	Ids []string
}

func newDownloadCommand() *cobra.Command {
	opt := downloadOptions{}
	cmd := &cobra.Command{
		Use:   "download --uuid=<uuid>|--alias=<alias>|--cluster=<clusterID>|<url>",
		Short: "download file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && opt.endpoint == "" {
				return cmd.Help()
			}

			if len(args) >= 1 {
				parseURL(&opt, args[0])
			}

			userName, password := credentials()
			opt.userName = userName
			opt.password = password
			opt.client = initClient(opt.endpoint)

			if opt.fileUUID != "" {
				return download(&opt)
			}

			if opt.fileAlias != "" {
				return downloadFilesByAlias(&opt)
			}

			if opt.clusterID > 0 {
				return downloadFilesByClusterID(&opt)
			}

			return errors.New("unsupport parameter")
		},
	}

	cmd.Flags().StringVarP(&opt.fileUUID, "uuid", "", "", "the uuid of file")
	cmd.Flags().StringVarP(&opt.fileAlias, "alias", "", "", "the alias of file")
	cmd.Flags().Uint64VarP(&opt.clusterID, "cluster-id", "", 0, "the cluster id of file")
	cmd.Flags().StringVarP(&opt.endpoint, "endpoint", "", "", "the clinic service endpoint.")

	return cmd
}

func parseURL(opt *downloadOptions, url string) {
	opt.fileUUID = url[strings.LastIndex(url, "/")+1:]

	opt.endpoint = url[:strings.Index(url, "/api")]
}

func download(opt *downloadOptions) error {
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
		log.Info("get current directory failed, use the default directory /tmp")
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

			err := downloadFilePart(job, tmpFile, opt)
			if err != nil {
				panic(fmt.Sprintf("download file part failed, err=%v", err))
			}
			log.Info("file part is done", zap.Int64("from", job.from), zap.Int64("to", job.to))
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

func fetchMeta(opt *downloadOptions) (*MetaData, error) {
	req, err := http.NewRequest(http.MethodHead, fmt.Sprintf("%s/api/internal/files/%s", opt.endpoint, opt.fileUUID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := requestWithAuth(&opt.clientOptions, req)
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

func downloadFilePart(part filePart, f *os.File, opt *downloadOptions) error {
	b := make([]byte, part.to-part.from)
	_, err := f.ReadAt(b, part.from)
	if err == nil {
		if bytes.Compare(make([]byte, part.to-part.from), b) != 0 {
			return nil
		}
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/%s", opt.endpoint, opt.fileUUID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", part.from, part.to))

	resp, err := requestWithAuth(&opt.clientOptions, req)
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

func downloadFilesByAlias(opt *downloadOptions) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/alias/%s", opt.endpoint, opt.fileAlias), nil)
	if err != nil {
		return err
	}

	resp, err := requestWithAuth(&opt.clientOptions, req)
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
		nopt := downloadOptions{
			clientOptions: opt.clientOptions,
			fileUUID:      id,
		}

		if err = download(&nopt); err != nil {
			return err
		}
	}

	return nil
}

func downloadFilesByClusterID(opt *downloadOptions) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/internal/files/cluster/%d", opt.endpoint, opt.clusterID), nil)
	if err != nil {
		return err
	}

	resp, err := requestWithAuth(&opt.clientOptions, req)
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
		nopt := downloadOptions{
			clientOptions: opt.clientOptions,
			fileUUID:      id,
		}

		if err = download(&nopt); err != nil {
			return err
		}
	}

	return nil
}
