package manager

import (
	"testing"

	. "github.com/pingcap/check"
)

func TestManager(t *testing.T) { TestingT(t) }

var _ = Suite(&testResolveSuite{})

type testResolveSuite struct{}
