package db

import (
	"os"
	"testing"
	"time"

	. "github.com/pingcap/check"
)

func TestGDB(t *testing.T) { TestingT(t) }

var _ = Suite(&testGDBSuite{})

type testGDBSuite struct {
	db DB
}

type userT struct {
	Name      string
	Age       int
	Birthday  time.Time
	IsStudent bool
}

type usert = userT

const TEST_DB = "/tmp/foresight-test.db"

func (s *testGDBSuite) SetUpTest(c *C) {
	var err error
	os.Remove(TEST_DB)
	s.db, err = Open(TEST_DB)
	c.Assert(err, IsNil)
	c.Assert(s.db.Table("users").CreateTable(&userT{}).Error(), IsNil)
	c.Assert(s.db.HasTable("users"), IsTrue)
	c.Assert(s.db.HasTable(&userT{}), IsFalse)
	c.Assert(s.db.CreateTable(&userT{}).Error(), IsNil)
	c.Assert(s.db.HasTable(&userT{}), IsTrue)
	c.Assert(s.db.HasTable(&usert{}), IsTrue)
}

func (s *testGDBSuite) TestCreate(c *C) {
	user := userT{
		Name:     "foresight-user",
		Age:      18,
		Birthday: time.Now(),
	}

	c.Assert(s.db.Table("users").Create(&user).Error(), IsNil)
}
