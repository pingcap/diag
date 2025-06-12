package parser

import (
	"time"

	json "github.com/json-iterator/go"
	"github.com/pingcap/diag/collector/log/item"
)

// Parse audit log in JSON format
//
// SET GLOBAL tidb_audit_log_format='JSON'
//
// Example (normally on a single line, but formatted for readability)
// {
//   "TIME": "2025/06/12 09:58:40.697 +02:00",
//   "ID": "988fbd53-7d09-40e9-95c5-8335c375ca9a-0007",
//   "EVENT": [
//     "QUERY",
//     "SELECT"
//   ],
//   "USER": "root",
//   "ROLES": [],
//   "CONNECTION_ID": "4083154952",
//   "SESSION_ALIAS": "",
//   "TABLES": [],
//   "STATUS_CODE": 1,
//   "CURRENT_DB": "test",
//   "SQL_TEXT": "select ?"
// }

type AuditJSONLogParser struct{}

func (*AuditJSONLogParser) ParseHead(head []byte) (*time.Time, item.LevelType) {
	if !json.Valid(head) {
		return nil, item.LevelInvalid
	}

	var headmap map[string]interface{}
	if err := json.Unmarshal(head, &headmap); err != nil {
		return nil, item.LevelInvalid
	}

	entryTime, ok := headmap["TIME"].(string)
	if !ok {
		return nil, item.LevelInvalid
	}
	t, err := parseTimeStamp([]byte(entryTime))
	if err != nil {
		return nil, item.LevelInvalid
	}

	return t, item.LevelInvalid
}
