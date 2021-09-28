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

type Decryptor interface {
	io.Reader
}

type decryptor struct {
	stream cipher.Stream
	buffer *bytes.Buffer
	reader io.Reader
}

func NewDecryptor(priv *rsa.PrivateKey, reader io.Reader) (Decryptor, error) {
	buffer := bytes.NewBuffer(nil)

	iv := make([]byte, priv.N.BitLen()/8)
	if n, err := reader.Read(iv); n != len(iv) || err != nil {
		return nil, fmt.Errorf("read buffer failed: %v, %d/%d bytes read", err, n, priv.N.BitLen()/8)
	}
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, iv, nil)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	return &decryptor{
		stream: cipher.NewCFBDecrypter(block, iv[:aes.BlockSize]),
		buffer: buffer,
		reader: reader,
	}, nil
}

func (e *decryptor) Read(p []byte) (n int, err error) {
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
