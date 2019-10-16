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
	testingDbFile, err := ioutil.TempFile("", "sqlite1.db")

	if err != nil {
		c.Fatal(err)
	}

	db, err := db.Open(testingDbFile.Name())
	if err != nil {
		c.Fatal(err)
	}

	s.db = db
	s.tmp = testingDbFile
	s.model = New(db)

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

func (s *testGDBSuite) TestingUpdateInspectionEstimateLeftSec(c *C) {
	idString := "114514"
	inspectionSample := Inspection{
		Uuid: idString,
	}
	s.db.Create(&inspectionSample)
	var inspectionQuery Inspection
	s.db.FirstOrCreate(&inspectionQuery)
	// default must be true
	c.Assert(inspectionQuery.EstimatedLeftSec == -1, IsTrue)

	var i int32
	for i = 0; i < 20; i++ {
		err := s.model.UpdateInspectionEstimateLeftSec(idString, i)
		if err != nil {
			// should not be nil
			c.Fatal(err)
		}
		users := make([]Inspection, 0)
		err = s.db.Find(&users).Error()
		if err != nil {
			c.Fatal("s.db.Find(&inspectionSample) error")
		}
		c.Assert(len(users) == 1, IsTrue)
		// default must be true
		c.Assert(users[0].EstimatedLeftSec == i, IsTrue)
	}

	i = -1000
	err := s.model.UpdateInspectionEstimateLeftSec(idString, i)
	if err == nil {
		// should be nil
		c.Fatal("UpdateInspectionEstimateLeftSec shouldn't accept arguments less than 1000")
	}
}
