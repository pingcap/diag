package dbinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/pingcap/tidb-foresight/model"
)

type Options interface {
	GetHome() string
	GetModel() model.Model
	GetInspectionId() string
	GetTidbStatusEndpoints() ([]string, error)
}

type DBInfoCollector struct {
	Options
}

func New(opts Options) *DBInfoCollector {
	return &DBInfoCollector{opts}
}

func (c *DBInfoCollector) Collect() error {
	endpoints, err := c.GetTidbStatusEndpoints()
	if err != nil {
		return err
	}

	for idx, endpoint := range endpoints {
		if schemas, err := c.schemaList(endpoint); err == nil {
			for _, schema := range schemas {
				if err := c.loadSchema(endpoint, schema); err != nil {
					return err
				}
			}
			return nil
		} else {
			if idx == len(endpoints)-1 {
				return err
			}
		}
	}

	return nil
}

func (c *DBInfoCollector) schemaList(endpoint string) ([]string, error) {
	schemas := []string{}

	resp, err := http.Get(fmt.Sprintf("http://%s/schema", endpoint))
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	r := []struct {
		DB struct {
			Name string `json:"L"`
		} `json:"db_name"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return []string{}, err
	}
	for _, schema := range r {
		schemas = append(schemas, schema.DB.Name)
	}
	return schemas, nil
}

func (c *DBInfoCollector) loadSchema(endpoint, schema string) error {
	home := c.GetHome()
	inspection := c.GetInspectionId()

	c.GetModel().UpdateInspectionMessage(inspection, fmt.Sprintf("collecting schema info for %s...", schema))

	resp, err := http.Get(fmt.Sprintf("http://%s/schema/%s", endpoint, schema))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	p := path.Join(home, "inspection", inspection, "dbinfo")
	if err := os.MkdirAll(p, os.ModePerm); err != nil {
		return err
	}

	dst, err := os.Create(path.Join(p, schema+".json"))
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	return err
}
