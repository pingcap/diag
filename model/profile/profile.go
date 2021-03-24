package profile

import (
	"os"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/utils"
	log "github.com/sirupsen/logrus"
)

const PROF_FILTER = "type = 'profile'"

type Profile struct {
	Uuid           string         `json:"uuid"`
	InstanceName   string         `json:"instance_name"`
	ClusterVersion string         `json:"cluster_version"`
	User           string         `json:"user"`
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	CreateTime     utils.NullTime `json:"create_time"`
	FinishTime     utils.NullTime `json:"finish_time"`
	Items          []ProfileItem  `json:"items"`
}

type ProfileItem struct {
	Component string   `json:"component"`
	Address   string   `json:"address"`
	Flames    []string `json:"flames"`
	Metas     []string `json:"metas"`
}

func fromInspection(insp *inspection.Inspection, profDir string) *Profile {
	prof := Profile{
		Uuid:           insp.Uuid,
		InstanceName:   insp.InstanceName,
		ClusterVersion: insp.ClusterVersion,
		User:           insp.User,
		Status:         insp.Status,
		Message:        insp.Message,
		CreateTime:     insp.CreateTime,
		FinishTime:     insp.FinishTime,
		Items:          []ProfileItem{},
	}

	prof.loadItems(profDir)
	return &prof
}

func (p *Profile) loadItems(dir string) {
	if p.Status != "success" {
		return
	}

	flist, err := os.ReadDir(path.Join(dir, p.Uuid))
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

		ms, err := p.listFileNames(path.Join(dir, p.Uuid, f.Name(), "meta"))
		if err != nil {
			log.Error("read profile meta:", err)
			continue
		}

		metas := []string{}
		for _, m := range ms {
			metas = append(metas, path.Join("/api", "v1", "perfprofiles", p.Uuid, xs[0], xs[1], "meta", m))
		}

		fs, err := p.listFileNames(path.Join(dir, p.Uuid, f.Name(), "flame"))
		if err != nil {
			log.Error("read profile flame:", err)
			continue
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
}

func (p *Profile) listFileNames(dir string) ([]string, error) {
	if files, err := os.ReadDir(dir); err != nil {
		return nil, err
	} else {
		names := []string{}
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names, nil
	}
}
