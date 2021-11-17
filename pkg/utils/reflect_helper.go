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

package utils

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

var DeprecatedTags = map[string]string{
	"enable-streaming": "EnableStreaming",
}

func ValueToInt(value reflect.Value) int64 {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(value.Uint())
	default:
		return 0
	}
}

func ValueToFloat(value reflect.Value) float64 {
	if value.Kind() == reflect.Float32 || value.Kind() == reflect.Float64 {
		return value.Float()
	}
	return 0
}

func ValueToBool(value reflect.Value) bool {
	if value.Kind() == reflect.Bool {
		return value.Bool()
	}
	return false
}

func ValueToString(value reflect.Value) string {
	if value.Kind() == reflect.String {
		return value.String()
	}
	return ""
}

func FlatMap(value reflect.Value, tagPath string) reflect.Value {
	if value.Kind() != reflect.Map {
		return reflect.ValueOf(nil)
	}
	if value.Len() == 0 {
		return reflect.ValueOf(nil)
	}
	mIter := value.MapRange()
	if len(tagPath) == 0 {
		result := reflect.MakeSlice(reflect.SliceOf(value.Type().Elem()), 0, value.Len())
		for mIter.Next() {
			result = reflect.Append(result, mIter.Value())
		}
		return result
	}
	tags := strings.Split(tagPath, ".")
	elem := VisitByTagPath(value.MapIndex(value.MapKeys()[0]), tags, 0)
	result := reflect.MakeSlice(reflect.SliceOf(elem.Type()), 0, value.Len())
	for mIter.Next() {
		result = reflect.Append(result, VisitByTagPath(mIter.Value(), tags, 0))
	}
	return result
}

func ElemInRange(value reflect.Value, l int64, h int64) bool {
	if value.Kind() != reflect.Slice {
		return false
	}
	switch value.Type().Elem().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		for i := 0; i < value.Len(); i++ {
			if value.Index(i).Int() < l || value.Index(i).Int() > h {
				return false
			}
		}
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		for i := 0; i < value.Len(); i++ {
			num := int64(value.Index(i).Uint())
			if num < l || num > h {
				return false
			}
		}
		return true
	case reflect.Float32, reflect.Float64:
		for i := 0; i < value.Len(); i++ {
			num := value.Index(i).Float()
			if num < float64(l) || num > float64(h) {
				return false
			}
		}
		return true
	}
	return false
}

func VisitByTagPath(node reflect.Value, tags []string, idx int) reflect.Value {
	if node.Interface() == nil {
		return reflect.ValueOf(nil)
	}
	value := node
	if value.Type().Kind() == reflect.Ptr {
		value = value.Elem()
	}
	valueType := value.Type()
	isLast := idx == len(tags)-1
	switch valueType.Kind() {
	case reflect.Struct:
		// check embedded field case first
		if valueType.NumField() == 1 && valueType.Field(0).Anonymous {
			return VisitByTagPath(value.Field(0), tags, idx)
		}
		// TODO:
		filedName, deprecated := DeprecatedTags[tags[idx]]
		if deprecated {
			field := value.FieldByName(filedName)
			if isLast {
				return field
			}
			return VisitByTagPath(field, tags, idx+1)
		}
		if isLast {
			for i := 0; i < valueType.NumField(); i++ {
				if jsonTagMatch(valueType.Field(i).Tag.Get("json"),tags[idx]) && value.Field(i).CanInterface() {
					return value.Field(i)
				}
			}
		} else {
			for i := 0; i < valueType.NumField(); i++ {
				if jsonTagMatch(valueType.Field(i).Tag.Get("json"),tags[idx]) {
					return VisitByTagPath(value.Field(i), tags, idx+1)
				}
			}
		}
	case reflect.Slice, reflect.Array:
		idx, err := parseIdx(tags[idx])
		if err != nil {
			return reflect.ValueOf(nil)
		}
		// panic if idx out of range
		if isLast {
			return value.Index(idx)
		}
		return VisitByTagPath(value.Index(idx), tags, idx+1)
	case reflect.Map:
		keyType := valueType.Key()
		switch keyType.Kind() {
		case reflect.String:
			mapValue := value.MapIndex(reflect.ValueOf(tags[idx]))
			if isLast {
				return mapValue
			}
			return VisitByTagPath(mapValue, tags, idx+1)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			k, err := strconv.Atoi(tags[idx])
			if err != nil {
				return reflect.ValueOf(nil)
			}
			if isLast {
				return value.MapIndex(reflect.ValueOf(k))
			}
			return VisitByTagPath(value.MapIndex(reflect.ValueOf(k)), tags, idx+1)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			k, err := strconv.ParseUint(tags[idx], 10, 64)
			if err != nil {
				return reflect.ValueOf(nil)
			}
			if isLast {
				return value.MapIndex(reflect.ValueOf(k))
			}
			return VisitByTagPath(value.MapIndex(reflect.ValueOf(k)), tags, idx+1)
		case reflect.Float64, reflect.Float32:
			k, err := strconv.ParseFloat(tags[idx], 10)
			if err != nil {
				return reflect.ValueOf(nil)
			}
			if isLast {
				return value.MapIndex(reflect.ValueOf(k))
			}
			return VisitByTagPath(value.MapIndex(reflect.ValueOf(k)), tags, idx+1)
		default:
			// not support return nil
			return reflect.ValueOf(nil)
		}
	}
	return reflect.ValueOf(nil) // which means not found any value match the given tag path
}

// @1 -> 1, @2 -> 2
func parseIdx(s string) (int, error) {
	if strings.HasPrefix(s, "@") {
		return 0, errors.New("index format error")
	}
	idxStr := strings.TrimPrefix(s, "@")
	return strconv.Atoi(idxStr)
}

func jsonTagMatch(tag string, target string) bool {
	if tag == target {
		return true
	}
	if strings.TrimSuffix(tag, ",string") == target {
		return true
	}
	if strings.TrimSuffix(tag, ",omitempty") == target {
		return true
	}
	if strings.TrimSuffix(tag, ",string,omitempty") == target {
		return true
	}
	return false
}
