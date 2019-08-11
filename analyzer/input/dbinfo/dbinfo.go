package dbinfo

// DBInfo contains information of all databases and tables
type DBInfo []*Database

// Database represents a database, which contains a list of tables
type Database struct {
	Name   string
	Tables []*Table
}

// The Table contains a list of indexes
// If a table contains no indexes, there should be a warning.
type Table struct {
	Name struct {
		L string `json:"L"`
	} `json:"name"`
	Indexes []struct {
		Id int `json:"id"`
	} `json:"index_info"`
}
