package profile

import (
	"io/ioutil"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
)

const PROF_FILTER = "type = 'profile'"

type Profile struct {
	Uuid         string         `json:"uuid"`
	InstanceName string         `json:"instance_name"`
	User         string         `json:"user"`
	Status       string         `json:"status"`
	CreateTime   utils.NullTime `json:"create_time"`
	FinishTime   utils.NullTime `json:"finish_time"`
	Items        []ProfileItem  `json:"items"`
}

type ProfileItem struct {
	Component string   `json:"component"`
	Address   string   `json:"address"`
	Flames    []string `json:"flames"`
	Metas     []string `json:"metas"`
}

func fromInspection(insp *inspection.Inspection, profDir string) (*Profile, error) {
	prof := Profile{
		Uuid:         insp.Uuid,
		InstanceName: insp.InstanceName,
		User:         insp.User,
		Status:       insp.Status,
		CreateTime:   insp.CreateTime,
		FinishTime:   insp.FinishTime,
	}

	return &prof, prof.loadItems(profDir)
}

func (p *Profile) loadItems(dir string) error {
	if p.Status != "success" {
		return nil
	}

	flist, err := ioutil.ReadDir(path.Join(dir, p.Uuid))
	if err != nil {
		return err
	}

	for _, f := range flist {
		// eg. pd-172.16.5.7:2379
		xs := strings.Split(f.Name(), "-")
		if len(xs) != 2 {
			// skip invalid directory name
			continue
		}

		ms, err := p.listFileNames(path.Join(dir, p.Uuid, f.Name(), "meta"))
		if err != nil {
			return err
		}

		metas := []string{}
		for _, m := range ms {
			metas = append(metas, path.Join("/api", "v1", "perfprofiles", p.Uuid, xs[0], xs[1], "meta", m))
		}

		fs, err := p.listFileNames(path.Join(dir, p.Uuid, f.Name(), "flame"))
		if err != nil {
			return err
		}

		flames := []string{}
		for _, f := range fs {
			flames = append(flames, path.Join("/api", "v1", "perfprofiles", p.Uuid, xs[0], xs[1], "flame", f))
		}

		p.Items = append(p.Items, ProfileItem{
			Component: xs[0],
			Address:   xs[1],
			Metas:     metas,
			Flames:    flames,
		})
	}

	return nil
}

func (p *Profile) listFileNames(dir string) ([]string, error) {
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
