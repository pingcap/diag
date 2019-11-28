package manager

import (
	"reflect"
	"sync"

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
		tm.outputs(t)
	}
	tm.current = len(tm.tasks)
}

//func (tm *TaskManager) ConcurrencyBatchRun(taskSz int) {
//	taskChan := make(chan *task, taskSz)
//
//	var wg sync.WaitGroup
//	for i := 1; i <= taskSz; i++ {
//		go func() {
//			for {
//				var currentTask *task
//				select {
//				case currentTask = <-taskChan:
//					log.Infof("counting task %v\n", currentTask.id)
//					tm.outputs(currentTask)
//					wg.Done()
//				default:
//					log.Infoln("return is called")
//					return
//				}
//			}
//		}()
//	}
//	log.Infof("current is %v, sum of len is %v\n", tm.current, len(tm.tasks))
//	wg.Add(len(tm.tasks) - tm.current)
//	//go func() {
//	//	for i, t := range tm.tasks[tm.current:len(tm.tasks)] {
//	//		fmt.Printf("Send task %v(%d)\n", t.id, i)
//	//		taskChan <- t
//	//	}
//	//}()
//
//	for i, t := range tm.tasks[tm.current:len(tm.tasks)] {
//		log.Infof("Send task %v(%d)\n", t.id, i + tm.current)
//		taskChan <- t
//	}
//	log.Infoln("Waiting now")
//	wg.Wait()
//	log.Infoln("close taskChan")
//	close(taskChan)
//
//	log.Infoln("RunCurrentBatch() finish batch [%v, %v)", tm.current, len(tm.tasks))
//
//	tm.current = len(tm.tasks)
//}

func (tm *TaskManager) ConcurrencyBatchRun(taskSz int) {
	var wg sync.WaitGroup
	wg.Add(len(tm.tasks) - tm.current)
	for _, t := range tm.tasks[tm.current:len(tm.tasks)] {
		go func() {
			log.Infof("counting task %v\n", t.id)
			tm.outputs(t)
			wg.Done()
		}()
	}
	wg.Wait()
	tm.current = len(tm.tasks)
}

func (tm *TaskManager) value(output string) reflect.Value {
	for _, t := range tm.tasks {
		// traverse all already output data.
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
	// checking arguments if using strict mode.
	if t.mode() == Strict {
		for _, arg := range args {
			if !arg.IsValid() || arg.IsNil() {
				return make([]reflect.Value, len(t.outputs))
			}
		}
	}
	return t.run(args)
}
