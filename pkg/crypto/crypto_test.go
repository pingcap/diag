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
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptAndDecrypt(t *testing.T) {
	assert := require.New(t)

	ciphertext := []byte("Hello, PingCAP")
	buffer := bytes.NewBuffer(ciphertext)

	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	assert.Nil(err)

	enc, err := NewEncryptor(&priv.PublicKey, buffer)
	assert.Nil(err)

	encBuf := make([]byte, 4096)
	n, err := enc.Read(encBuf)
	assert.True(n > 0)
	assert.Nil(err)

	nn, err := enc.Read(encBuf[n:])
	assert.Nil(err)
	assert.True(n > 0)

	dec, err := NewDecryptor(priv, bytes.NewBuffer(encBuf[:n+nn]))
	assert.Nil(err)

	decBuf := make([]byte, 4096)
	n, err = dec.Read(decBuf)
	assert.Nil(err)
	assert.True(n > 0)

	assert.Equal("Hello, PingCAP", string(decBuf[:n]))
}
