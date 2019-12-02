package dbinfo

import (
	"encoding/json"
	"github.com/pingcap/tidb-foresight/utils/debug_printer"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pingcap/tidb-foresight/analyzer/boot"
	log "github.com/sirupsen/logrus"
)

type parseDBInfoTask struct{}

func ParseDBInfo() *parseDBInfoTask {
	return &parseDBInfoTask{}
}

// Parse all {db}.json, they are outputs of TiDB query:
// http://{TiDBIP}:10080/schema/{db}
func (t *parseDBInfoTask) Run(c *boot.Config) *DBInfo {
	dbinfo := DBInfo{}

	dbs, err := ioutil.ReadDir(path.Join(c.Src, "dbinfo"))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("read dir:", err)
		}
		return nil
	}

	for _, db := range dbs {
		// skip invalid file and special db
		if !strings.HasSuffix(db.Name(), ".json") || strings.HasSuffix(db.Name(), "_schema.json") {
			continue
		}
		tbs, err := parseTables(path.Join(c.Src, "dbinfo", db.Name()))
		if err != nil {
			log.Error("parse tables:", err)
			return nil
		}
		dbinfo = append(dbinfo, &Database{
			Name:   strings.TrimSuffix(db.Name(), ".json"),
			Tables: tbs,
		})
	}
	log.Infof("load dbinfo %v", debug_printer.FormatJson(dbinfo))
	return &dbinfo
}

func parseTables(file string) ([]*Table, error) {
	tables := []*Table{}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err = json.NewDecoder(f).Decode(&tables); err != nil {
		return nil, err
	}

	return tables, nil
}
