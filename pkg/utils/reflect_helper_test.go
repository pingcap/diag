// Copyright 2023 PingCAP, Inc.
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

package utils

import (
	"reflect"
	"testing"
)

func TestVal2Int(t *testing.T) {
	tt := []struct {
		input  interface{}
		output int64
	}{
		{input: int(-123), output: -123},
		{input: int32(-123), output: -123},
		{input: int64(-123), output: -123},
		{input: uint(123), output: 123},
		{input: uint32(123), output: 123},
		{input: uint64(123), output: 123},
		{input: "123", output: 0},
	}

	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			val := reflect.ValueOf(tc.input)
			out := ValueToInt(val)
			if out != tc.output {
				t.Fatalf("got %v, expect %v", out, tc.output)
			}
		})
	}
}

func TestVal2Str(t *testing.T) {
	tt := []struct {
		input  reflect.Value
		output string
	}{
		{input: reflect.ValueOf(int(-123)), output: "-123"},
		{input: reflect.ValueOf(uint(123)), output: "123"},
		{input: reflect.ValueOf("123"), output: "123"},
		{
			input:  reflect.ValueOf([]int{-111, 222, 333}),
			output: "-111,222,333",
		},
		{
			input:  reflect.ValueOf([]uint{111, 222, 333}),
			output: "111,222,333",
		},
		{
			input:  reflect.ValueOf([]string{"-111", "222", "333"}),
			output: "-111,222,333",
		},
	}

	for _, tc := range tt {
		t.Run("", func(t *testing.T) {
			out := ValueToString(tc.input)
			if out != tc.output {
				t.Fatalf("got %v, expect %v", out, tc.output)
			}
		})
	}
}
