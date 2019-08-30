package utils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type TagdString struct {
	value string
	tags  url.Values
}

func NewTagdString(value string, tags map[string]string) TagdString {
	tv := TagdString{value: value, tags: url.Values{}}
	for k, v := range tags {
		tv.tags.Set(k, v)
	}
	return tv
}

func (tv *TagdString) GetValue() string {
	return tv.value
}

func (tv *TagdString) SetValue(value string) {
	tv.value = value
}

func (tv *TagdString) GetTag(key string) string {
	return tv.tags.Get(key)
}

func (tv *TagdString) SetTag(key, value string) {
	tv.tags.Set(key, value)
}

// Scan implements the Scanner interface.
func (tv *TagdString) Scan(value interface{}) error {
	// The format of value should be value,tag_key1=tag_val1,tag_key2=tag_val2...
	str, ok := value.(string)
	if !ok {
		return nil
	}

	tags := strings.Split(str, ",")

	if len(tags) < 1 {
		return nil
	}

	if v, err := url.QueryUnescape(tags[0]); err != nil {
		return err
	} else {
		tv.value = v
	}

	if len(tags) < 2 {
		return nil
	}

	if ts, err := url.ParseQuery(tags[1]); err != nil {
		return err
	} else {
		tv.tags = ts
		return nil
	}
}

// Value implements the driver Valuer interface.
func (tv TagdString) Value() (driver.Value, error) {
	vs := []string{url.QueryEscape(tv.value), tv.tags.Encode()}

	return strings.Join(vs, ","), nil
}

// MarshalJSON implements the json.Marshaler
func (tv TagdString) MarshalJSON() ([]byte, error) {
	if len(tv.tags) == 0 {
		return json.Marshal(tv.value)
	}

	m := make(map[string]interface{}, 0)
	for k, vs := range tv.tags {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	m["value"] = tv.value
	m["abnormal"] = true // TODO: remove this
	return json.Marshal(m)
}

type TagdFloat64 struct {
	TagdString
}

func NewTagdFloat64(value float64, tags map[string]string) TagdFloat64 {
	return TagdFloat64{
		NewTagdString(fmt.Sprintf("%f", value), tags),
	}
}

func (tv *TagdFloat64) GetValue() float64 {
	if v, err := strconv.ParseFloat(tv.TagdString.GetValue(), 64); err != nil {
		log.Error("parse float64 value:", err)
		return 0
	} else {
		return v
	}
}

func (tv *TagdFloat64) SetValue(value float64) {
	tv.TagdString.SetValue(fmt.Sprintf("%f", value))
}

// MarshalJSON implements the json.Marshaler
func (tv TagdFloat64) MarshalJSON() ([]byte, error) {
	if len(tv.tags) == 0 {
		return json.Marshal(tv.value)
	}

	m := make(map[string]interface{}, 0)
	for k, vs := range tv.tags {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	m["value"] = tv.value
	m["abnormal"] = true // TODO: remove this
	return json.Marshal(m)
}
