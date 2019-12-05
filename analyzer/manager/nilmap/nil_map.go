package nilmap

import (
	"reflect"
)

func init() {
	tolerate = make(map[string]struct{})
}

// This method can only be used in `init` method.
// It register the class in tolerate map.
func TolerateRegister(v reflect.Type) {
	tolerate[tpName(v)] = struct{}{}
}

func TolerateRegisterStruct(o interface{}) {
	var valStruct reflect.Type

	if o == nil {
		panic("Arguments in TolerateRegisterStruct should not be nil")
	}
	t := reflect.TypeOf(o)

	if t.Kind() == reflect.Ptr {
		valStruct = t.Elem()
	} else {
		valStruct = t
	}

	if valStruct.Kind() != reflect.Struct {
		panic("Arguments in TolerateRegisterStruct should not be struct")
	}
	TolerateRegister(valStruct)
}

func IsTolerate(tpName string) bool {
	_, isTor := tolerate[tpName]
	return isTor
}

// `tolerate` is a set for storing name for object.
var tolerate map[string]struct{}

func tpName(t reflect.Type) string {
	helper := func(tp reflect.Type) string {
		return tp.PkgPath() + "#" + tp.Name()
	}
	if t.Kind() != reflect.Ptr {
		return helper(t)
	}
	valOfStr := t.Elem()
	return helper(valOfStr)
}
