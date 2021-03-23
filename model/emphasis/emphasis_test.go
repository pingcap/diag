package emphasis

import (
	"database/sql"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/pingcap/check"
	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
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
	testingDbFile, err := ioutil.TempFile("", "sqlite.db")

	if err != nil {
		c.Fatal(err)
	}

	testingDb, err := db.Open(testingDbFile.Name())
	if err != nil {
		c.Fatal(err)
	}

	s.db = testingDb.Debug()
	s.tmp = testingDbFile
	s.model = New(testingDb.Debug())

	c.Assert(err, IsNil)

	s.db.CreateTable(&inspection.Inspection{})
	s.db.CreateTable(&Problem{})
	c.Assert(s.db.HasTable(&inspection.Inspection{}), IsTrue)
	c.Assert(s.db.HasTable(&Problem{}), IsTrue)
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

	emps, _, err := s.model.ListAllEmphasis(0, 5)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(len(emps) == 1, IsTrue)
	c.Assert(emps[0].Uuid == emp.Uuid, IsTrue)

	emp2, err := s.model.GetEmphasis("1321")

	if err != nil {
		c.Fatal(err)
	}
	c.Assert(emp2.Uuid == emp.Uuid, IsTrue)

	newProb := &Problem{
		Problem:      utils.NullString{sql.NullString{}},
		Uuid:         "2103712",
		RelatedGraph: "tidb",
		Advise:       "suicide",
	}

	err = s.model.AddProblem(emp.Uuid, newProb)
	if err != nil {
		c.Fatal(err)
	}

	probs, err := s.model.LoadAllProblems(emp)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(len(probs) == 1, IsTrue)
	c.Assert(probs[0].Uuid == newProb.Uuid, IsTrue)

	// Create inspection with other Types other than emphasis.
	s.db.Create(inspection.Inspection{})

	emps, _, err = s.model.ListAllEmphasis(0, 5)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(len(emps) == 1, IsTrue)

	empk := &Emphasis{
		Uuid:              "1325",
		CreatedTime:       time.Now(),
		InvestgatingEnd:   time.Now(),
		InvestgatingStart: time.Now(),
	}
	err = s.model.CreateEmphasis(empk)
	if err != nil {
		c.Fatal(err)
	}

	err = s.model.DeleteEmphasis("1321")
	if err != nil {
		c.Fatal(err)
	}

	emps, _, err = s.model.ListAllEmphasis(0, 5)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(len(emps) == 1, IsTrue)
	c.Assert(empk.Uuid == "1325", IsTrue)
}
