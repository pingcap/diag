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
	if o == nil {
		panic("Arguments in TolerateRegisterStruct should not be nil")
	}
	t := reflect.TypeOf(o)
	if t.Kind() != reflect.Ptr {
		panic("Arguments in TolerateRegisterStruct should not be str")
	}
	valOfStr := t.Elem()
	if valOfStr.Kind() != reflect.Struct {
		panic("Arguments in TolerateRegisterStruct should not be struct")
	}
	TolerateRegister(t)
}

func IsTolerate(tpName string) bool {
	_, isTor := tolerate[tpName]
	return isTor
}

// tolerate is a set for
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
