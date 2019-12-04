package manager

import (
	"reflect"
	"testing"

	. "github.com/pingcap/check"
)

func TestManager(t *testing.T) { TestingT(t) }

var _ = Suite(&testResolveSuite{})

type testResolveSuite struct{}


func TestIsEmptyStruct(t *testing.T) {
	type EmptySample struct{}

	if !isEmptyStruct(reflect.TypeOf(EmptySample{})) && !isEmptyStruct(reflect.TypeOf(&EmptySample{})) {
		t.Error("isEmptyStruct(EmptySample{}) return true")
	}

	type NotEmpty struct {
		val float32
	}

	if isEmptyStruct(reflect.TypeOf(NotEmpty{})) && isEmptyStruct(reflect.TypeOf(&NotEmpty{})) {
		t.Error("isEmptyStruct(NotEmpty{}) return false")
	}

	iptr := 5
	if isEmptyStruct(reflect.TypeOf(iptr)) {
		t.Error("isEmptyStruct(&5) return true")
	}

	var emptyPtr *EmptySample
	emptyPtr = nil

	if isEmptyStruct(reflect.TypeOf(emptyPtr)) {
		t.Error("isEmptyStruct() panic on pointer")
	}
}

type emptyDone struct{}
type Task1 struct{}
type Task2 struct{}
type TaskRequireDone struct{}

func (t *Task1) Run() *emptyDone {
	return &emptyDone{}
}

func (t *TaskRequireDone) Run(e *emptyDone) {

}

func TestValidTest(t *testing.T) {
	manager := New()
	manager.tasks = append(manager.tasks, newTask(&Task1{}, Strict))
	manager.tasks = append(manager.tasks, newTask(&TaskRequireDone{}, Strict))

	manager.Run()
}
