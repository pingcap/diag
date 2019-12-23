package nilmap

import (
	"reflect"
	"testing"

	. "github.com/pingcap/check"
)

// Registered type
type Tp1 struct{}

// Unregistered type
type Tp2 struct{}

func init() {
	TolerateRegister(reflect.TypeOf(&Tp1{}))
}

func TestNilMap(t *testing.T) { TestingT(t) }

type testNilMapSuite struct{}

var _ = Suite(&testNilMapSuite{})

// Only for testing, please not using it outside!!!
func tp(o interface{}) reflect.Type {
	return reflect.TypeOf(o)
}

func (s *testNilMapSuite) TestNilMap(c *C) {
	c.Assert(IsTolerate(tpName(tp(Tp1{}))), IsTrue)
	tp1Ptr := &Tp1{}
	c.Assert(IsTolerate(tpName(tp(tp1Ptr))), IsTrue)
	c.Assert(IsTolerate(tpName(tp(Tp2{}))), IsFalse)
	c.Assert(IsTolerate(tpName(tp(&Tp2{}))), IsFalse)
}
