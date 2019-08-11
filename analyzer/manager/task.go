package manager

import (
	"reflect"
)

// The task is an abstraction of analyze task
// It must work with TaskManager
type task struct {
	// The id identity the path and name of the task: {pkg_path}#{type_name}
	id string

	// The method stored the reflect value of Run method in target type
	method reflect.Value

	// The inputs stored the types of input list of the Run method
	// The format of each inputs element is {pkg_path}#{type_name}
	// eg. github.com/pingcap/tidb-foresight/analyzer/boot#DB
	inputs []string

	// The outputs stored the types of output list of the Run method
	// The format of each inputs element is {pkg_path}#{type_name}
	// eg. github.com/pingcap/tidb-foresight/analyzer/boot#DB
	outputs []string

	// The cache field cached the values returned by last execution of
	// this task, it's used to guarantee every task run atmost once
	cache []reflect.Value
}

// Genrate a task struct from any struct pointer having Run method
// The Run method is required, however, it's signature is not important,
// the input list of the Run method will be filled by TaskManager, the
// output list of the Run method will be collected for other taks' input
// list.
// eg.
// type Task1 struct {}
// type Task2 struct {}
// type Task1Out struct {}
//
// func (*Task1) Run() *Task1Out {
//		return &Task1Out{}
// }
//
// func (Task2) Run(*Task1Out) {
//		// something here
// }
//
// The TaskManager will guarantee the Task1's Run method
// will be called before Task2's, no matter how the register
// to the TaskManager.
func newTask(i interface{}) *task {
	t := &task{}

	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		panic("task is not a pointer:" + v.Type().PkgPath() + "#" + v.Type().Name())
	}

	t.id = v.Elem().Type().PkgPath() + "#" + v.Elem().Type().Name()
	t.method = v.MethodByName("Run")
	if !t.method.IsValid() {
		panic(t.id + " does't have method Run")
	}

	for idx := 0; idx < t.method.Type().NumIn(); idx++ {
		if t.method.Type().In(idx).Kind() != reflect.Ptr {
			panic("only support ptr as input args at present, check Run method of task " + t.id)
		}
		typeId := t.method.Type().In(idx).Elem().PkgPath() + "#" + t.method.Type().In(idx).Elem().Name()
		t.inputs = append(t.inputs, typeId)
	}

	for idx := 0; idx < t.method.Type().NumOut(); idx++ {
		if t.method.Type().Out(idx).Kind() != reflect.Ptr {
			panic("only support ptr as output at present, check Run method of task " + t.id)
		}
		typeId := t.method.Type().Out(idx).Elem().PkgPath() + "#" + t.method.Type().Out(idx).Elem().Name()
		t.outputs = append(t.outputs, typeId)
	}

	return t
}

// Call method with args and return result of Call
func (t *task) run(args []reflect.Value) []reflect.Value {
	if len(t.cache) != 0 {
		return t.cache
	}
	for idx, arg := range args {
		if arg == reflect.ValueOf(nil) {
			// If required input not found, use nil pointer
			args[idx] = reflect.Zero(t.method.Type().In(idx))
		}
	}
	t.cache = t.method.Call(args)
	return t.cache
}

type ResolveMode int16

const (
	Strict ResolveMode = iota
	Tolerance
)
