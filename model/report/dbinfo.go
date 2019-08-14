package report

import (
	log "github.com/sirupsen/logrus"
)

type DBInfo struct {
	DB    string `json:"schema"`
	Table string `json:"table"`
	Index int    `json:"index"`
}

func (r *Report) loadDBInfo() error {
	if !r.itemReady("dbinfo") {
		return nil
	}

	rows, err := r.db.Query(
		`SELECT db, tb, idx from inspection_db_info WHERE inspection = ?`,
		r.inspectionId,
	)
	if err != nil {
		log.Error("db.Query: ", err)
		return err
	}

	dbinfo := []*DBInfo{}
	for rows.Next() {
		info := DBInfo{}
		err = rows.Scan(&info.DB, &info.Table, &info.Index)
		if err != nil {
			log.Error("db.Query:", err)
			return err
		}

		dbinfo = append(dbinfo, &info)
	}

	r.DBInfo = dbinfo
	return nil
}
