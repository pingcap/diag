package statement

type Config struct {
	TiDBRefreshInterval int `json:"tidb_refresh_interval"`
	MaxStmtCount		int `json:"max_stmt_count"`
	MaxSQLLength		int `json:"max_sql_length"`
	StmtCountLimit		int `json:"stmt_count_limit"`
}

func (m *statement) GetStatementConfig(instanceId string) (*Config, error) {
	return nil, nil 
}

func (m *statement) SetStatementConfig(instance, cfg *Config) error {
	return nil
}