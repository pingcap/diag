package task

import (
	"path"
	"io/ioutil"
	"encoding/json"
	"database/sql"
	log "github.com/sirupsen/logrus"
)

type DBInfo []*Database

type Database struct {
	Name string
	Tables []*Table
}

type Table struct {
	Name struct {
		L string `json:"L"`
	} `json:"name"`
	Indexes []struct {
		Id int `json:"id"`
	} `json:"index_info"`
}

type ParseDBInfoTask struct {
	BaseTask
}


type SaveDBInfoTask struct {
	BaseTask
}

func ParseDBInfo(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &ParseDBInfoTask {BaseTask{inspectionId, src, data, db}}
}

func SaveDBInfo(inspectionId string, src string, data *TaskData, db *sql.DB) Task {
	return &SaveDBInfoTask {BaseTask{inspectionId, src, data, db}}
}

func (t *ParseDBInfoTask) Run() error {
	if !t.data.collect[ITEM_DBINFO] || t.data.status[ITEM_DBINFO].Status != "success" {
		return nil
	}

	dbinfo := DBInfo{}

	dbs, err := ioutil.ReadDir(path.Join(t.src, "dbinfo"))
	if err != nil {
		log.Error("read dir: ", err)
		return err
	}

	for _, db := range dbs {
		tbs, err := parseTables(path.Join(t.src, "dbinfo", db.Name()))
		if err != nil {
			log.Error("parse tables: ", err)
			return err
		}
		dbinfo = append(dbinfo, &Database{
			Name: db.Name(),
			Tables: tbs,
		})
	}

	t.data.dbinfo = dbinfo

	return nil
}

func parseTables(dir string) ([]*Table, error) {
	tables := []*Table{}

	tbs, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Error("read dir: ", err)
		return tables, err
	}

	for _, tb := range tbs {
		tb, err := parseTable(path.Join(dir, tb.Name()))
		if err != nil {
			log.Error("parse table: ", err)
			return tables, err
		}
		tables = append(tables, tb)
	}

	return tables, nil
}

func parseTable(file string) (*Table, error) {
	table := Table{}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error("read file: ", err)
		return nil, err
	}

	if err = json.Unmarshal(content, &table); err != nil {
		log.Error("unmarshal: ", err)
		return nil, err
	}

	return &table, nil
}

func (t *SaveDBInfoTask) Run() error {
	if !t.data.collect[ITEM_DBINFO] || t.data.status[ITEM_DBINFO].Status != "success" {
		return nil
	}

	for _, schema := range t.data.dbinfo {
		for _, tb := range schema.Tables {
			if _, err := t.db.Exec(
				"REPLACE INTO inspection_db_info(inspection, db, tb, idx) VALUES(?, ?, ?, ?)",
				t.inspectionId, schema.Name, tb.Name.L, len(tb.Indexes),
			); err != nil {
				log.Error("db.Exec: ", err)
				return err
			}
		}
	}
	return nil
}