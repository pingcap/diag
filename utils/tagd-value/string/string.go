package string

import (
	"database/sql/driver"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

type String struct {
	value string
	tags  url.Values
}

func New(value string, tags map[string]string) String {
	tv := String{value: value, tags: url.Values{}}
	for k, v := range tags {
		tv.tags.Set(k, v)
	}
	return tv
}

func (tv *String) GetValue() string {
	return tv.value
}

func (tv *String) SetValue(value string) {
	tv.value = value
}

func (tv *String) GetTag(key string) string {
	return tv.tags.Get(key)
}

func (tv *String) SetTag(key, value string) {
	tv.tags.Set(key, value)
}

// Scan implements the Scanner interface.
func (tv *String) Scan(value interface{}) error {
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
func (tv String) Value() (driver.Value, error) {
	vs := []string{url.QueryEscape(tv.value), tv.tags.Encode()}

	return strings.Join(vs, ","), nil
}

// Return all tags
func (tv *String) Tags() map[string]string {
	m := make(map[string]string, 0)
	for k, vs := range tv.tags {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

// MarshalJSON implements the json.Marshaler
func (tv String) MarshalJSON() ([]byte, error) {
	if len(tv.Tags()) == 0 {
		return jsoniter.Marshal(tv.GetValue())
	}

	m := make(map[string]interface{}, 0)
	for k, v := range tv.Tags() {
		m[k] = v
	}
	m["value"] = tv.GetValue()
	return jsoniter.Marshal(m)
}
