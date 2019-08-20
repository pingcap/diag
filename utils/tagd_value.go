package utils

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type TagdString struct {
	V    string
	Tags url.Values
}

func NewTagdString(value string, tags map[string]string) TagdString {
	tv := TagdString{V: value, Tags: url.Values{}}
	for k, v := range tags {
		tv.Tags.Set(k, v)
	}
	return tv
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
	tv.V = tags[0]

	if len(tags) < 2 {
		return nil
	}

	if ts, err := url.ParseQuery(tags[1]); err != nil {
		return err
	} else {
		tv.Tags = ts
		return nil
	}
}

// Value implements the driver Valuer interface.
func (tv TagdString) Value() (driver.Value, error) {
	vs := []string{tv.V, tv.Tags.Encode()}

	return strings.Join(vs, ","), nil
}

type TagdFloat64 struct {
	TagdString
	V float64
}

func NewTagdFloat64(value float64, tags map[string]string) TagdFloat64 {
	return TagdFloat64{
		NewTagdString(fmt.Sprintf("%f", value), tags),
		value,
	}
}

// Scan implements the Scanner interface.
func (tv *TagdFloat64) Scan(value interface{}) error {
	if err := tv.TagdString.Scan(value); err != nil {
		return err
	}

	if v, err := strconv.ParseFloat(tv.TagdString.V, 64); err != nil {
		return err
	} else {
		tv.V = v
		return nil
	}
}

// Value implements the driver Valuer interface.
func (tv TagdFloat64) Value() (driver.Value, error) {
	tv.TagdString.V = fmt.Sprintf("%f", tv.V)

	return tv.TagdString.Value()
}
