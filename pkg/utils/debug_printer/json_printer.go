package debugprinter

import (
	"github.com/fatih/structs"
	json "github.com/json-iterator/go"
)

// panic if marshall return nil
func FormatJSON(object interface{}) string {
	data, err := json.MarshalIndent(object, "", "    ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

//JSONMapper maps the object
func JSONMapper(object interface{}) map[string]interface{} {
	return structs.Map(object)
}
