package inspection

import (
	"io/ioutil"
	"os"

	"testing"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb-foresight/wrapper/db"
)

func TestUpdateTest(t *testing.T) { TestingT(t) }

var _ = Suite(&testGDBSuite{})

type testGDBSuite struct {
	db    db.DB
	tmp   *os.File
	model Model
}

func (s *testGDBSuite) SetUpTest(c *C) {
	testingDbFile, err := ioutil.TempFile("", "sqlite-test.db")

	if err != nil {
		c.Fatal(err)
	}

	testingDb, err := db.Open(testingDbFile.Name())
	if err != nil {
		c.Fatal(err)
	}

	s.db = testingDb
	s.tmp = testingDbFile
	s.model = New(testingDb)

	c.Assert(err, IsNil)
	s.db.CreateTable(&Inspection{})
	c.Assert(s.db.HasTable(&Inspection{}), IsTrue)
	c.Assert(s.db.HasTable(&Inspection{}), IsTrue)
}

func (s *testGDBSuite) TearDownTest(c *C) {
	err := s.db.Close()
	if err != nil {
		c.Fatal(err)
	}
	err = s.tmp.Close()
	if err != nil {
		c.Fatal(err)
	}
	err = os.Remove(s.tmp.Name()) // clean up
	if err != nil {
		c.Fatal(err)
	}
}
