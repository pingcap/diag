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

type Encryptor interface {
	io.Reader
}

type encryptor struct {
	stream cipher.Stream
	buffer *bytes.Buffer
	reader io.Reader
}

func NewEncryptor(pub *rsa.PublicKey, reader io.Reader) (Encryptor, error) {
	buffer := bytes.NewBuffer(nil)

	aesKey := make([]byte, 32)
	n, err := rand.Read(aesKey)
	if n != len(aesKey) || err != nil {
		return nil, fmt.Errorf("generate aes key failed: %v, %d/%d bytes generated", err, n, len(aesKey))
	}
	iv, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey[:], nil)
	if err != nil {
		return nil, err
	}
	if n, err := buffer.Write(iv); n != len(iv) || err != nil {
		return nil, fmt.Errorf("write buffer failed: %v, %d/%d bytes write", err, n, len(iv))
	}
	block, err := aes.NewCipher(aesKey[:])
	if err != nil {
		return nil, err
	}

	return &encryptor{
		stream: cipher.NewCFBEncrypter(block, iv[:aes.BlockSize]),
		buffer: buffer,
		reader: reader,
	}, nil
}

func (e *encryptor) Read(p []byte) (n int, err error) {
	if e.buffer.Len() == 0 {
		inBuf := make([]byte, BufferSize)
		n, err := e.reader.Read(inBuf)
		if n == 0 {
			return 0, err
		}

		outBuf := make([]byte, n)
		e.stream.XORKeyStream(outBuf, inBuf[:n])
		if n, err := e.buffer.Write(outBuf); n != len(outBuf) || err != nil {
			return 0, fmt.Errorf("write buffer failed: %v, %d/%d bytes write", err, n, len(outBuf))
		}
	}

	return e.buffer.Read(p)
}
