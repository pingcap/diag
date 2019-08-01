package model

import (
	"io/ioutil"
	"path"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Profile struct {
	Uuid         string        `json:"uuid"`
	InstanceName string        `json:"instance_name"`
	Status       string        `json:"status"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Items        []ProfileItem `json:"items"`
}

type ProfileItem struct {
	Component string   `json:"component"`
	Address   string   `json:"address"`
	Flames    []string `json:"flames"`
	Metas     []string `json:"metas"`
}

func (p *Profile) loadItems(dir string) error {
	flist, err := ioutil.ReadDir(path.Join(dir, p.Uuid))
	if err != nil {
		log.Error("read dir: ", err)
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
			log.Error("list dir:", err)
			return err
		}

		metas := []string{}
		for _, m := range ms {
			metas = append(metas, path.Join("/api", "v1", "perfprofiles", p.Uuid, xs[0], xs[1], "meta", m))
		}

		fs, err := p.listFileNames(path.Join(dir, p.Uuid, f.Name(), "flame"))
		if err != nil {
			log.Error("list dir:", err)
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

func (m *Model) ListAllProfiles(page, size int64, profileDir string) ([]*Profile, int, error) {
	profiles := []*Profile{}

	rows, err := m.db.Query(
		`SELECT id,instance,status,create_t,create_t FROM inspections WHERE id IN (
			SELECT inspection FROM inspection_items WHERE name = 'profile' AND status <> 'none'
		) limit ?,?`,
		(page-1)*size, size,
	)
	if err != nil {
		log.Error("failed to call db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		profile := Profile{}
		if err := rows.Scan(&profile.Uuid, &profile.InstanceName, &profile.Status, &profile.StartTime, &profile.EndTime); err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}
		if err = profile.loadItems(profileDir); err != nil {
			log.Error("load profile items:", err)
			return nil, 0, err
		}
		profiles = append(profiles, &profile)
	}

	total := 0
	if err = m.db.QueryRow(
		"SELECT COUNT(DISTINCT(inspection)) FROM inspection_items WHERE name = 'profile' AND status <> 'none'",
	).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return profiles, total, nil
}

func (m *Model) ListProfiles(instanceId string, page, size int64, profileDir string) ([]*Profile, int, error) {
	profiles := []*Profile{}

	rows, err := m.db.Query(
		`SELECT id,instance,status,create_t,create_t FROM inspections 
		WHERE instance = ? AND id IN (
			SELECT inspection FROM inspection_items WHERE status <> 'none'
		) limit ?,?`,
		instanceId, (page-1)*size, size,
	)
	if err != nil {
		log.Error("Failed to call db.Query:", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		profile := Profile{}
		err := rows.Scan(&profile.Uuid, &profile.InstanceName, &profile.Status, &profile.StartTime, &profile.EndTime)
		if err != nil {
			log.Error("db.Query:", err)
			return nil, 0, err
		}
		if err = profile.loadItems(profileDir); err != nil {
			log.Error("load profile items:", err)
			return nil, 0, err
		}
		profiles = append(profiles, &profile)
	}

	total := 0
	if err = m.db.QueryRow(
		`SELECT COUNT(id) FROM inspections 
		 WHERE instance = ? AND id IN (
			SELECT inspection FROM inspection_items WHERE status <> 'none'
		)`,
		instanceId,
	).Scan(&total); err != nil {
		log.Error("db.Query:", err)
		return nil, 0, err
	}

	return profiles, total, nil
}

func (m *Model) GetProfileDetail(profileId, profileDir string) (*Profile, error) {
	profile := Profile{}
	if err := m.db.QueryRow(
		`SELECT id,instance,status,create_t,create_t FROM inspections WHERE id = ?`,
		profileId,
	).Scan(
		&profile.Uuid, &profile.InstanceName, &profile.Status, &profile.StartTime, &profile.EndTime,
	); err != nil {
		log.Error("db.Query:", err)
		return nil, err
	}

	if err := profile.loadItems(profileDir); err != nil {
		log.Error("load profile items:", err)
		return nil, err
	}

	return &profile, nil
}