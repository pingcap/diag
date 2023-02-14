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

package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"
)

type EncryptWriter struct {
	stream cipher.Stream
	header *bytes.Buffer
	w      io.Writer
}

func NewEncryptWriter(pub *rsa.PublicKey, w io.Writer) (*EncryptWriter, error) {
	header := bytes.NewBuffer(nil)

	aesKey := make([]byte, 32)
	n, err := rand.Read(aesKey)
	if n != len(aesKey) || err != nil {
		return nil, fmt.Errorf("generate aes key failed: %v, %d/%d bytes generated", err, n, len(aesKey))
	}
	iv, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey[:], nil)
	if err != nil {
		return nil, err
	}
	if n, err := header.Write(iv); n != len(iv) || err != nil {
		return nil, fmt.Errorf("write buffer failed: %v, %d/%d bytes write", err, n, len(iv))
	}
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}

	return &EncryptWriter{
		stream: cipher.NewCFBEncrypter(block, iv[:aes.BlockSize]),
		header: header,
		w:      w,
	}, nil
}

func (w *EncryptWriter) Write(p []byte) (n int, err error) {
	var headn int64
	// write OAEP header at first write
	if w.header.Len() > 0 {
		headn, err := io.Copy(w.w, w.header)
		if err != nil {
			return int(headn), err
		}
	}
	outBuf := make([]byte, len(p))
	w.stream.XORKeyStream(outBuf, p)
	n, err = w.w.Write(outBuf)
	return int(headn) + n, err
}
