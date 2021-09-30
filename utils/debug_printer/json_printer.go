package debugprinter

import (
	"github.com/fatih/structs"
	jsoniter "github.com/json-iterator/go"
)

// panic if marshall return nil
func FormatJSON(object interface{}) string {
	data, err := jsoniter.MarshalIndent(object, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

//JSONMapper maps the object
func JSONMapper(object interface{}) map[string]interface{} {
	return structs.Map(object)
}
