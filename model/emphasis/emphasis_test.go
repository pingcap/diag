package emphasis

import (
	"github.com/pingcap/tidb-foresight/model/inspection"
	"testing"
	"time"
)

import (
	"io/ioutil"
	"os"

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
	testingDbFile, err := ioutil.TempFile(".", "sqlite.db")

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

	s.db.CreateTable(&inspection.Inspection{})
	c.Assert(s.db.HasTable(&inspection.Inspection{}), IsTrue)
	c.Assert(s.db.HasTable(&inspection.Inspection{}), IsTrue)
}

func (s *testGDBSuite) TestingCreate(c *C) {
	emp := &Emphasis{
		Uuid:              "1321",
		CreatedTime:       time.Now(),
		InvestgatingEnd:   time.Now(),
		InvestgatingStart: time.Now(),
	}
	err := s.model.CreateEmphasis(emp)
	if err != nil {
		c.Fatal(err)
	}

	emps, _, err := s.model.ListAllEmphasis(5, 5)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(len(emps) == 1, IsTrue)
	c.Assert(emps[0] == emp, IsTrue)

	emp2, err := s.model.GetEmphasis("1321")
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(emp2 == emp, IsTrue)
}
