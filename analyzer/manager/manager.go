package manager

import (
	"reflect"
)

// The TaskManager's responsibility is to maintain a list of
// tasks and find proper inputs for each tasks and run them.
type TaskManager struct {
	// The mode indicate if downstream task should be executed
	// if upstream broken
	mode  ResolveMode
	tasks []*task
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		mode:  Strict,
		tasks: make([]*task, 0),
	}
}

// Set resolve mode
func (tm *TaskManager) Mode(mode ResolveMode) *TaskManager {
	tm.mode = mode
	return tm
}

func (tm *TaskManager) Register(tasks ...interface{}) *TaskManager {
	for _, t := range tasks {
		tm.tasks = append(tm.tasks, newTask(t))
	}
	return tm
}

func (tm *TaskManager) Run() {
	for _, t := range tm.tasks {
		tm.outputs(t)
	}
}

func (tm *TaskManager) value(output string) reflect.Value {
	for _, t := range tm.tasks {
		for idx, o := range t.outputs {
			if o == output {
				return tm.outputs(t)[idx]
			}
		}
	}

	// required value not found
	return reflect.ValueOf(nil)
}

func (tm *TaskManager) outputs(t *task) []reflect.Value {
	args := make([]reflect.Value, len(t.inputs))
	for i := 0; i < len(args); i++ {
		args[i] = tm.value(t.inputs[i])
	}

	if tm.mode == Strict {
		for _, arg := range args {
			if !arg.IsValid() || arg.IsNil() {
				return make([]reflect.Value, len(t.outputs))
			}
		}
	}
	return t.run(args)
}
