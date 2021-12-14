package float64

import (
	"fmt"
	"strconv"

	json "github.com/json-iterator/go"
	ts "github.com/pingcap/diag/pkg/utils/tagd-value/string"
	log "github.com/sirupsen/logrus"
)

type Float64 struct {
	ts.String
}

func New(value float64, tags map[string]string) Float64 {
	return Float64{
		ts.New(fmt.Sprintf("%f", value), tags),
	}
}

func (tv *Float64) GetValue() float64 {
	v, err := strconv.ParseFloat(tv.String.GetValue(), 64)
	if err != nil {
		log.Error("parse float64 value:", err)
		return 0
	}
	return v
}

func (tv *Float64) SetValue(value float64) {
	tv.String.SetValue(fmt.Sprintf("%f", value))
}

// MarshalJSON implements the json.Marshaler
func (tv Float64) MarshalJSON() ([]byte, error) {
	if len(tv.Tags()) == 0 {
		return json.Marshal(tv.GetValue())
	}

	m := make(map[string]interface{}, 0)
	for k, v := range tv.Tags() {
		m[k] = v
	}
	m["value"] = tv.GetValue()
	return json.Marshal(m)
}
