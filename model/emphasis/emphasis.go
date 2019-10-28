package emphasis

import (
	"io/ioutil"
	"path"
	"strings"
	"time"

	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

type Emphasis struct {
	Uuid              string    `json:"uuid"`
	InstanceId        string    `json:"instance_id"`
	CreatedTime       time.Time `json:"created_time"`
	InvestgatingStart time.Time `json:"investgating_start"`
	InvestgatingEnd   time.Time `json:"investgating_end"`

	InvestgatingProblem string `json:"investgating_problem"`

	RelatedProblems []Problem `json:"related_problems" gorm:"foreignkey:UserRefer"`
}

func (emp *Emphasis) CorrespondInspection() *inspection.Inspection {
	return &inspection.Inspection{
		Uuid:        emp.Uuid,
		InstanceId:  emp.InstanceId,
		CreateTime:  utils.FromTime(emp.CreatedTime),
		ScrapeBegin: utils.FromTime(emp.InvestgatingStart),
		ScrapeEnd:   utils.FromTime(emp.InvestgatingEnd),

		Type: "emphasis",
	}
}

func InspectionToEmphasis(insp *inspection.Inspection) *Emphasis {
	return &Emphasis{
		Uuid:              insp.Uuid,
		InstanceId:        insp.InstanceId,
		CreatedTime:       insp.CreateTime.Time,
		InvestgatingStart: insp.ScrapeBegin.Time,
		InvestgatingEnd:   insp.ScrapeEnd.Time,
	}
}

func (emp *Emphasis) loadProblems(dir string) {

	flist, err := ioutil.ReadDir(path.Join(dir, emp.Uuid))
	if err != nil {
		log.Error("read profile directory:", err)
		return
	}

	for _, f := range flist {
		// eg. pd-172.16.5.7:2379
		xs := strings.Split(f.Name(), "-")
		if len(xs) != 2 {
			// skip invalid directory name
			continue
		}

		ms, err := emp.listFileNames(path.Join(dir, emp.Uuid, f.Name(), "meta"))
		if err != nil {
			log.Error("read profile meta:", err)
			continue
		}

		metas := []string{}
		for _, m := range ms {
			metas = append(metas, path.Join("/api", "v1", "perfprofiles", emp.Uuid, xs[0], xs[1], "meta", m))
		}

		fs, err := emp.listFileNames(path.Join(dir, emp.Uuid, f.Name(), "flame"))
		if err != nil {
			log.Error("read profile flame:", err)
			continue
		}

		flames := []string{}
		for _, f := range fs {
			flames = append(flames, path.Join("/api", "v1", "perfprofiles", emp.Uuid, xs[0], xs[1], "flame", f))
		}

		// TODO: how to link here?
		emp.RelatedProblems = append(emp.RelatedProblems, Problem{})
	}
}

func (emp *Emphasis) listFileNames(dir string) ([]string, error) {
	if files, err := ioutil.ReadDir(dir); err != nil {
		return nil, err
	} else {
		names := []string{}
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names, nil
	}
}
