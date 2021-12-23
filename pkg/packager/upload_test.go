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
	"fmt"
	"io"
	"math"
	"sync"
	"testing"

	"github.com/onsi/gomega"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
)

func Test_ComputeTotalBlock(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	v := computeTotalBlock(60, 20)
	g.Expect(v).To(gomega.Equal(3))

	v = computeTotalBlock(60, 21)
	g.Expect(v).To(gomega.Equal(3))
}

func Test_UploadInitFile(t *testing.T) {
	blockSize := 6

	contents := make([]byte, 0)
	for i := 0; i < 100; i++ {
		contents = append(contents, byte(i))
	}

	resp := &preCreateResponse{
		Partseq:    0,
		BlockBytes: int64(blockSize),
	}

	g := gomega.NewGomegaWithT(t)

	mt := &mapTest{
		results: make(map[int64][]byte),
	}

	logger := logprinter.NewLogger("")
	_, err := UploadFile(logger, resp, int64(len(contents)),
		flushFunc(g, resp, contents, mt),
		func() (io.ReadSeekCloser, error) {
			reader := NewMockReader(contents, int(resp.BlockBytes))
			return reader, nil
		},
		uploadFunc(g, mt, len(contents), blockSize))

	g.Expect(err).To(gomega.Succeed())
}

func Test_UploadFileMultiBlock(t *testing.T) {
	blockSize := 6

	contents := make([]byte, 0)
	for i := 0; i < 96; i++ {
		contents = append(contents, byte(i))
	}

	g := gomega.NewGomegaWithT(t)
	for i := 0; i < 16; i++ {
		resp := &preCreateResponse{
			Partseq:    i,
			BlockBytes: int64(blockSize),
		}

		mt := &mapTest{
			results: make(map[int64][]byte),
		}

		logger := logprinter.NewLogger("")
		_, err := UploadFile(logger, resp, int64(len(contents)),
			flushFunc(g, resp, contents, mt),
			func() (io.ReadSeekCloser, error) {
				reader := NewMockReader(contents, int(resp.BlockBytes))
				return reader, nil
			},
			uploadFunc(g, mt, len(contents), blockSize))

		g.Expect(err).To(gomega.Succeed())
	}
}

func Test_UploadFileFromOffset(t *testing.T) {
	blockSize := 6

	contents := make([]byte, 0)
	for i := 0; i < 100; i++ {
		contents = append(contents, byte(i))
	}

	g := gomega.NewGomegaWithT(t)
	for i := 1; i < 17; i++ {
		resp := &preCreateResponse{
			Partseq:    i,
			BlockBytes: int64(blockSize),
		}

		mt := &mapTest{
			results: make(map[int64][]byte),
		}

		logger := logprinter.NewLogger("")
		_, err := UploadFile(logger, resp, int64(len(contents)),
			flushFunc(g, resp, contents, mt),
			func() (io.ReadSeekCloser, error) {
				reader := NewMockReader(contents, int(resp.BlockBytes))
				return reader, nil
			},
			uploadFunc(g, mt, len(contents), blockSize))

		g.Expect(err).To(gomega.Succeed())
	}
}

var uploadFunc = func(g *gomega.WithT, mt *mapTest, fileSize int, defaultBlockSize int) UploadPart {
	return func(i, size int64, r io.Reader) error {
		buf := make([]byte, size)
		r.Read(buf)
		mt.put(i, buf)
		totalBlock := computeTotalBlock(int64(fileSize), int64(defaultBlockSize))
		if i == int64(totalBlock) {
			if fileSize%defaultBlockSize == 0 {
				g.Expect(int(size)).To(gomega.Equal(defaultBlockSize))
			} else {
				g.Expect(int(size)).To(gomega.Equal(fileSize % defaultBlockSize))
			}

		} else {
			g.Expect(int(size)).To(gomega.Equal(defaultBlockSize))
		}
		return nil
	}
}

var flushFunc = func(g *gomega.WithT, resp *preCreateResponse, contents []byte, mt *mapTest) FlushUploadFile {
	return func() (string, error) {
		fmt.Println("\n<>>>>>>>>>")

		totalBlockSize := computeTotalBlock(int64(len(contents)), resp.BlockBytes)

		g.Expect(len(mt.results)).To(gomega.Equal(totalBlockSize - resp.Partseq))

		for i := resp.Partseq; i < totalBlockSize; i++ {
			actualData := mt.results[int64(i+1)]

			offset := i * int(resp.BlockBytes)
			end := math.Min(float64(offset+int(resp.BlockBytes)), float64(len(contents)))

			expectData := contents[offset:int(end)]

			g.Expect(len(actualData)).To(gomega.Equal(len(expectData)))

			for i := 0; i < len(expectData); i++ {
				g.Expect(expectData[i]).To(gomega.Equal(actualData[i]))
			}
		}

		fmt.Println("Complete!")
		return "", nil
	}
}

type mapTest struct {
	lock    sync.Mutex
	results map[int64][]byte
}

func (m *mapTest) put(serial int64, data []byte) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.results[serial] = data
}

type MockReader struct {
	reader *bytes.Reader

	blockSize int
	closed    bool
}

func NewMockReader(contents []byte, blockSize int) *MockReader {
	reader := bytes.NewReader(contents)

	return &MockReader{
		reader:    reader,
		blockSize: blockSize,
	}
}

func (m *MockReader) Close() error {
	m.closed = true

	return nil
}

func (m *MockReader) Read(b []byte) (int, error) {
	return m.reader.Read(b)
}

func (m *MockReader) Seek(offset int64, whence int) (int64, error) {
	return m.reader.Seek(offset, whence)
}
