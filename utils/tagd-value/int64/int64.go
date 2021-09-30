package int64

import (
	"fmt"
	"strconv"

	jsoniter "github.com/json-iterator/go"
	ts "github.com/pingcap/diag/utils/tagd-value/string"
	log "github.com/sirupsen/logrus"
)

type Int64 struct {
	ts.String
}

func New(value int64, tags map[string]string) Int64 {
	return Int64{
		ts.New(fmt.Sprintf("%d", value), tags),
	}
}

func (tv *Int64) GetValue() int64 {
	v, err := strconv.ParseInt(tv.String.GetValue(), 10, 64)
	if err != nil {
		log.Error("parse int64 value:", err)
		return 0
	}
	return v
}

func (tv *Int64) SetValue(value int64) {
	tv.String.SetValue(fmt.Sprintf("%d", value))
}

// MarshalJSON implements the json.Marshaler
func (tv Int64) MarshalJSON() ([]byte, error) {
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
