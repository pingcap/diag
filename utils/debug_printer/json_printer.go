package debug_printer

import (
	"github.com/fatih/structs"
	jsoniter "github.com/json-iterator/go"
)

// panic if marshall return nil
func FormatJson(object interface{}) string {
	data, err := jsoniter.MarshalIndent(object, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func JsonMapper(object interface{}) map[string]interface{} {
	return structs.Map(object)
}
