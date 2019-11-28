package statement

type StatementRange struct {
	Begin string `json:"begin"`
	End   string `json:"end"`
}

func (m *statement) ListSchemas(instanceId string) ([]string, error) {
	stmts := []*Statement{}
	schemas := []string{}

	if err := m.db.Raw(
		"SELECT DISTINCT(SCHEMA_NAME) AS SCHEMA_NAME WHERE INSTANCE_ID = ?",
		instanceId,
	).Find(&stmts).Error(); err != nil {
		return nil, err
	}
	for _, stmt := range stmts {
		schemas = append(schemas, stmt.SchemaName)
	}

	return schemas, nil
}

func (m *statement) ListSchemaStatementRange(instanceId, schema string) ([]*StatementRange, error) {
	stmts := []*Statement{}
	rgs := []*StatementRange{}

	if err := m.db.Raw(
		"SELECT DISTINCT SUMMARY_BEGIN_TIME, SUMMARY_END_TIME",
	).Find(&stmts).Error(); err != nil {
		return nil, err
	}
	for _, stmt := range stmts {
		rgs = append(rgs, &StatementRange{
			Begin: stmt.SummaryBeginTime,
			End:   stmt.SummaryEndTime,
		})
	}

	return rgs, nil
}
