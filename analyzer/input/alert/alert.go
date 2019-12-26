package alert

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// The alert information from prometheus
type Alert []struct {
	Metric Metric `json:"metric"`
	Value []interface{}
}

type Metric struct {
	Name       string `json:"alertname"`
	Alertstate string `json:"alertstate"`
	Env        string `json:"env"`
	Expr       string `json:"expr"`
	Level      string `json:"level"`
	Req        string `json:"req"`
	Instance   string `json:"instance"`

	ExtraInfo map[string]interface{} `json:"detail,omitempty"`
}

func unmarshalJsonObject(jsonStr []byte, obj interface{}, otherFields map[string]json.RawMessage) (err error) {
	objValue := reflect.ValueOf(obj).Elem()
	knownFields := map[string]reflect.Value{}
	for i := 0; i != objValue.NumField(); i++ {
		jsonName := strings.Split(objValue.Type().Field(i).Tag.Get("json"), ",")[0]
		knownFields[jsonName] = objValue.Field(i)
	}

	err = json.Unmarshal(jsonStr, &otherFields)
	if err != nil {
		return
	}

	for key, chunk := range otherFields {
		if field, found := knownFields[key]; found {
			err = json.Unmarshal(chunk, field.Addr().Interface())
			if err != nil {
				return
			}
			delete(otherFields, key)
		}
	}
	return
}

func (metric *Metric) UnmarshalJSON(jsonStr []byte) (err error)  {
	fmt.Println("Unmarshal")
	other := map[string]json.RawMessage{}
	err = unmarshalJsonObject(jsonStr, metric, other)
	if err != nil {
		return err
	}
	metric.ExtraInfo = make(map[string]interface{})
	for k, v := range other {
		data, err := v.MarshalJSON()
		if err != nil {
			return err
		}
		metric.ExtraInfo[k] = string(data)
	}
	return nil
}