// Copyright 2022 PingCAP, Inc.
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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAndParserD1agHeader(t *testing.T) {
	assert := require.New(t)

	meta := map[string]interface{}{
		"cluster_id":   "12345678123456781234567812345678",
		"cluster_type": "tidb-cluster",
		"ext":          "To boldly go where no one has gone before",
	}
	file, err := GenerateD1agHeader(meta, TypeTar, "")
	if err != nil {
		fmt.Println("--------------")
		fmt.Println(err)
	}
	assert.Nil(err)

	metabyte, encryption, compress, offset, err := ParserD1agHeader(bytes.NewBuffer(file))
	assert.Nil(err)
	assert.EqualValues("none", encryption)
	assert.EqualValues("tar", compress)
	assert.EqualValues(len(file), offset)

	meta2 := make(map[string]interface{})
	err = json.Unmarshal(metabyte, &meta2)
	assert.Nil(err)
	assert.EqualValues(meta, meta2)

}
