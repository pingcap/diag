package debug_printer

import (
	"encoding/json"

	"github.com/fatih/structs"
)

// panic if marshall return nil
func FormatJson(object interface{}) string {
	data, err := json.MarshalIndent(object, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func JsonMapper(object interface{}) map[string]interface{} {
	return structs.Map(object)
}
