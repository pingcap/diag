package logs

import (
	"strings"

	"github.com/pingcap/tidb-foresight/model/inspection"
	"github.com/pingcap/tidb-foresight/model/instance"
)

type LogEntity struct {
	Id           string `json:"uuid"`
	InstanceName string `json:"instance_name"`
}

func (m *logs) ListLogFiles(ids []string) ([]*LogEntity, error) {
	ls := []*LogEntity{}

	if len(ids) == 0 {
		return ls, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	insps := []*inspection.Inspection{}
	filter := "type = 'log'"

	if err := m.db.Where(filter).Find(&insps).Error(); err != nil {
		return nil, err
	}

	for _, insp := range insps {
		// FIXME: this is not required?
		if !strings.Contains(idstr, insp.Uuid) {
			continue
		}
		l := LogEntity{
			Id:           insp.Uuid,
			InstanceName: insp.InstanceName,
		}
		ls = append(ls, &l)
	}

	return ls, nil
}

func (m *logs) ListLogInstances(ids []string) ([]*LogEntity, error) {
	ls := []*LogEntity{}

	if len(ids) == 0 {
		return ls, nil
	}

	idstr := `"` + strings.Join(ids, `","`) + `"`

	insts := []*instance.Instance{}
	if err := m.db.Find(&insts).Error(); err != nil {
		return nil, err
	}

	for _, inst := range insts {
		// FIXME: this is not required?
		if !strings.Contains(idstr, inst.Uuid) {
			continue
		}
		l := LogEntity{
			Id:           inst.Uuid,
			InstanceName: inst.Name,
		}
		ls = append(ls, &l)
	}

	return ls, nil
}
