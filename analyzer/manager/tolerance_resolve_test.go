package manager

import (
	//"testing"

	. "github.com/pingcap/check"
	"testing"
)

func (s *testResolveSuite) TestToleranceResolve(c *C) {
	ta := &tTaskA{}
	tb := &tTaskB{c, false}
	tc := &tTaskC{c, false}
	td := &tTaskD{c, 0, false}
	te := &tTaskE{c, Strict}
	New().Mode(Tolerance).Register(
		td,
		te,
		tc,
		tb,
		ta,
	).Run()
	c.Assert(tb.runFlag, IsTrue)
	c.Assert(tc.runFlag, IsTrue)
}

type tTaskA struct{}
type tOutputOfA1 struct{}
type tOutputOfA2 struct{}

func (*tTaskA) Run() (*tOutputOfA1, *tOutputOfA2) {
	return &tOutputOfA1{}, nil
}

type tTaskB struct {
	c       *C
	runFlag bool
}

func (t *tTaskB) Run(i *tOutputOfA1) {
	t.runFlag = true
	t.c.Assert(i, NotNil)
}

type tTaskC struct {
	c       *C
	runFlag bool
}

func (t *tTaskC) Run(i *tOutputOfA2) {
	t.runFlag = true
	t.c.Assert(i, IsNil)
}

type tTaskD struct {
	c       *C
	Mode    int64
	runFlag bool
}

func (t *tTaskD) Run(i *tOutputOfA2) {
	t.runFlag = true
	t.c.Assert(i, IsNil)
}

type tTaskE struct {
	c    *C
	Mode ResolveMode
}

func (t *tTaskE) Run(i *tOutputOfA2) {
	t.c.Error("this task should not run")
}

func TestIsEmptyStruct(t *testing.T) {
	type EmptySample struct{}

	if !isEmptyStruct(EmptySample{}) {
		t.Error("isEmptyStruct(EmptySample{}) return true")
	}

	type NotEmpty struct {
		val float32
	}

	if isEmptyStruct(NotEmpty{}) {
		t.Error("isEmptyStruct(NotEmpty{}) return false")
	}
}
