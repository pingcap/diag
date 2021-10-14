package command

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"golang.org/x/term"
	"hash/fnv"
	"strings"
	"syscall"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"go.uber.org/zap"

	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
	"github.com/spf13/cobra"
)

type preCreateResponse struct {
	Partseq int
	BlockBytes int64
}

type uploadOptions struct {
	filePath string
	alias string
	clientOptions
}

type clientOptions struct {
	endpoint string
	userName string
	password string
	client *http.Client
}

func newUploadCommand() *cobra.Command {
	opt := uploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload <file>",
		Short: "upload a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return cmd.Help()
			}
			opt.filePath = args[0]

			userName, password := credentials()
			opt.userName = userName
			opt.password = password
			opt.client = initClient(opt.endpoint)

			return upload(&opt)
		},
	}

	cmd.Flags().StringVarP(&opt.alias, "alias", "", "", "the alias of upload file.")
	cmd.Flags().StringVarP(&opt.endpoint, "endpoint", "", "", "the clinic service endpoint.")
	cmd.MarkFlagRequired("endpoint")

	return cmd
}

func upload(opt *uploadOptions) error {
	fileStat, err := os.Stat(opt.filePath)
	if err != nil {
		return err
	}

	uuid := fmt.Sprintf("%s-%s-%s", fnvHash(fileStat.Name()), fnvHash(fmt.Sprintf("%d", fileStat.Size())), fnvHash(fileStat.ModTime().Format(time.RFC3339)))
	if opt.alias != "" {
		uuid = fmt.Sprintf("%s-%s", uuid, fnvHash32(opt.alias))
	}

	presp, err := preCreate(uuid, fileStat.Size(), fileStat.Name(), opt)
	if err != nil {
		return err
	}

	return UploadFile(presp, fileStat.Size(), func() error {
		return UploadComplete(uuid, opt)
	}, func() (ReaderAtCloseable, error) {
		return os.Open(opt.filePath)
	}, func(serial int64, data []byte) error {
		return uploadMultipartFile(uuid, serial, data, opt)
	})
}

func preCreate(uuid string, fileLen int64, originalName string, opt *uploadOptions) (*preCreateResponse, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/precreate", opt.endpoint), nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("uuid", uuid)
	q.Add("fileLen", fmt.Sprintf("%d", fileLen))
	q.Add("alias", opt.alias)
	q.Add("orignalName", originalName)
	req.URL.RawQuery = q.Encode()

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.clientOptions, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
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

type ReaderAtCloseable interface {
	Close() error

	ReadAt(b []byte, off int64) (n int, err error)
}

type Reader func() (ReaderAtCloseable, error)

type UploadPart func(int64, []byte) error

type FlushUploadFile func() error

func UploadFile(presp *preCreateResponse, fileSize int64, flush FlushUploadFile, readerFunc Reader, uploadPart UploadPart) error {
	totalBlock := computeTotalBlock(fileSize, presp.BlockBytes)
	if totalBlock <= presp.Partseq {
		return flush()
	}

	reader, err := readerFunc()
	if err != nil {
		return err
	}
	defer reader.Close()

	waitGroup := sync.WaitGroup{}
	for i := int64(presp.Partseq); i < int64(totalBlock); i++ {
		eachSize := presp.BlockBytes
		if i == int64(totalBlock)-1 {
			eachSize = fileSize - i*presp.BlockBytes
		}

		s := make([]byte, eachSize)
		n, _ := reader.ReadAt(s, i*presp.BlockBytes)

		if n < 0 {
			fmt.Fprintf(os.Stderr, "cat: error reading: %s\n", err.Error())
			os.Exit(1)
		}

		if n > 0 {
			waitGroup.Add(1)
			go func(serial int64) {
				defer waitGroup.Done()
				fmt.Printf(">>>>>>>>>")

				if err = uploadPart(serial, s); err != nil {
					fmt.Fprintf(os.Stderr, "cat: upload file failed: %s\n", err.Error())
					os.Exit(1)
				}

				fmt.Printf(">>>>>>>>>")
			}(i + 1)
		}
	}

	waitGroup.Wait()
	return flush()
}

func appendClinicHeader(req *http.Request) {
	req.Header.Add("x-clinic-client", "upload")
}

func UploadComplete(fileUUID string, opt *uploadOptions) error {
	fmt.Println("\n<>>>>>>>>>")
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/flush", opt.endpoint), nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("uuid", fileUUID)
	req.URL.RawQuery = q.Encode()

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.clientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("upload file failed, msg=%v", string(body))
	}

	fmt.Println("Completed!")
	fmt.Printf("Download URL: %s\n", string(body))

	if his, err := loadHistroy(); err == nil {
		his.Push(string(body))
		his.Store()
	}

	return nil
}

func fnvHash(raw string) string {
	hash := fnv.New64()
	if _, err := hash.Write([]byte(raw)); err != nil {
		// impossible path
		log.Error("failed to write fnv hash", zap.Error(err))
	}
	return fmt.Sprintf("%x", hash.Sum64())
}

func fnvHash32(raw string) string {
	hash := fnv.New32()
	if _, err := hash.Write([]byte(raw)); err != nil {
		// impossible path
		log.Error("failed to write fnv hash", zap.Error(err))
	}
	return fmt.Sprintf("%x", hash.Sum32())
}

func uploadMultipartFile(fileUUID string, serialNum int64, data []byte, opt *uploadOptions) error {
	if len(data) == 0 {
		return nil
	}
	var b bytes.Buffer
	mwriter := multipart.NewWriter(&b)

	fw, err := mwriter.CreateFormFile("file", "file")
	if err != nil {
		return err
	}

	fw.Write(data)

	mwriter.Close()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/v1/upload", opt.endpoint), &b)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Add("uuid", fileUUID)
	q.Add("partseq", fmt.Sprintf("%d", serialNum))
	q.Add("partlen", fmt.Sprintf("%d", len(data)))
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Content-Type", mwriter.FormDataContentType())

	appendClinicHeader(req)
	resp, err := requestWithAuth(&opt.clientOptions, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("upload file failed, msg=%v", string(body))
	}

	return nil
}

func initClient(endpoint string) *http.Client{
	if strings.HasPrefix(strings.ToLower(endpoint), "https://") {
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM([]byte(rootCA))
		if !ok {
			panic("failed to parse root certificate")
		}
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: roots},
			},
		}

	}

	return http.DefaultClient
}

var rootCA = `-----BEGIN CERTIFICATE-----
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

func credentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}

	password := string(bytePassword)
	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func requestWithAuth(opt *clientOptions, req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(opt.userName, opt.password)
	resp, err := opt.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("Unauthorized")
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("Some requests can only be used on the Internal network")
	}

	if resp.StatusCode == http.StatusBadRequest {
		defer resp.Body.Close()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.New(" Bad Request")
		}

		errmsg := resp.Status
		if string(data) != "" {
			errmsg = fmt.Sprintf("%s. %s", errmsg, string(data))
		}
		return nil, errors.Errorf(" Bad Request, msg=%s", errmsg)
	}

	if resp.StatusCode == http.StatusProcessing {
		return nil, errors.New("the resource is processing, please again later")
	}

	return resp, nil
}