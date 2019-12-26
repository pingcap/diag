package manager

import (
	"reflect"
	"sync"

	"github.com/pingcap/tidb-foresight/analyzer/manager/nilmap"
	log "github.com/sirupsen/logrus"
)

// The TaskManager's responsibility is to maintain a list of
// tasks and find proper inputs for each tasks and run them.
type TaskManager struct {
	// The mode indicate if downstream task should be executed
	// if upstream broken
	mode  ResolveMode
	tasks []*task

	// The cursor of the current executing task.
	// Will be initialized as 0
	current int
}

func New() *TaskManager {
	return &TaskManager{
		mode:    Strict,
		tasks:   make([]*task, 0),
		current: 0,
	}
}

// Set resolve mode
func (tm *TaskManager) Mode(mode ResolveMode) *TaskManager {
	tm.mode = mode
	return tm
}

func (tm *TaskManager) Register(tasks ...interface{}) *TaskManager {
	for _, t := range tasks {
		tm.tasks = append(tm.tasks, newTask(t, tm.mode))
	}
	return tm
}

func (tm *TaskManager) Run() {
	tm.current = 0
	tm.RunCurrentBatch()
}

func (tm *TaskManager) RunCurrentBatch() {
	for _, t := range tm.tasks[tm.current:] {
		log.Infof("RunCurrentBatch calling %s", t.id)
		tm.outputs(t)
	}
	log.Infof("RunCurrentBatch runs from %v to %v", tm.current, len(tm.tasks))
	tm.current = len(tm.tasks)
}

func (tm *TaskManager) ConcurrencyBatchRun(taskSz int) {
	taskChan := make(chan *task, taskSz)

	var wg sync.WaitGroup

	log.Infof("ConcurrencyBatchRun current is %v, sum of len is %v\n", tm.current, len(tm.tasks))
	wg.Add(len(tm.tasks) - tm.current)

	go func() {
		for _, t := range tm.tasks[tm.current:len(tm.tasks)] {
			taskChan <- t
		}
	}()

	for i := 1; i <= taskSz; i++ {
		go func() {
			for {
				select {
				case currentTask, closed := <-taskChan:
					if !closed {
						return
					}
					tm.outputs(currentTask)
					wg.Done()
				}
			}
		}()
	}
	wg.Wait()
	close(taskChan)
	log.Infof("ConcurrencyRunBatch runs from %v to %v", tm.current, len(tm.tasks))
	tm.current = len(tm.tasks)
}

func (tm *TaskManager) value(output string) reflect.Value {
	for _, t := range tm.tasks {
		// traverse all already output data or drives posterior tasks to run.
		for idx, o := range t.outputs {
			if o == output {
				return tm.outputs(t)[idx]
			}
		}
	}

	// required value not found
	log.Warnf("DEBUG: `value` arm not match output %v, return `reflect.ValueOf(nil)`.", output)
	return reflect.ValueOf(nil)
}

func (tm *TaskManager) outputs(t *task) []reflect.Value {
	args := make([]reflect.Value, len(t.inputs))
	for i := 0; i < len(args); i++ {
		log.Infof("%s Collecting arguments: ", t.id)

		// waiting for all arguments
		args[i] = tm.value(t.inputs[i])

		log.Infof("%s Collecting arguments done", t.id)
	}
	// checking arguments if using strict mode.
	if t.mode() == Strict {
		for i, arg := range args {
			if !arg.IsValid() || arg.IsNil() {

				if nilmap.IsTolerate(t.inputs[i]) {
					log.Warn("argument %v(%d) in task %v is invalid, but is is registered tolerate," +
						" default arguments was setting here.")
					continue
				}

				// In strict mode, if argument is invalid, fill a argument here.
				log.Warnf("argument %v(%d) in task %v is invalid, default arguments was setting here.",
					arg, i, t)
				return make([]reflect.Value, len(t.outputs))
			}
		}
	}
	return t.run(args)
}
