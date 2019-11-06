package manager

import (
	. "github.com/pingcap/check"
)

func (s *testResolveSuite) TestStrictResolve(c *C) {
	ta := &sTaskA{}
	tb := &sTaskB{c, false}
	tc := &sTaskC{c}
	td := &sTaskD{c, 0}
	te := &sTaskE{c, Tolerance, false}
	New().Mode(Strict).Register(
		td,
		tc,
		tb,
		ta,
		te,
	).Run()
	c.Assert(tb.runFlag, IsTrue)
	c.Assert(te.runFlag, IsTrue)
}

type sTaskA struct{}
type sOutputOfA1 struct{}
type sOutputOfA2 struct{}

func (*sTaskA) Run() (*sOutputOfA1, *sOutputOfA2) {
	return &sOutputOfA1{}, nil
}

type sTaskB struct {
	c       *C
	runFlag bool
}

func (t *sTaskB) Run(i *sOutputOfA1) {
	t.runFlag = true
	t.c.Assert(i, NotNil)
}

type sTaskC struct {
	c *C
}

func (t *sTaskC) Run(i *sOutputOfA2) {
	t.c.Error("this task should not run")
}

type sTaskD struct {
	c    *C
	Mode int64
}

func (t *sTaskD) Run(i *sOutputOfA2) {
	t.c.Error("this task should not run")
}

type sTaskE struct {
	c       *C
	Mode    ResolveMode
	runFlag bool
}

func (t *sTaskE) Run(i *sOutputOfA2) {
	t.c.Assert(i, IsNil)
	t.runFlag = true
}
