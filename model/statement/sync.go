package statement

import (
	"time"
)

// Sync statement data from TiDB to sqlite.db
func (m *statement) StatementSync(instanceId string) error {
	// Get the refresh interval to decide which data to synchronize
	interval, err := m.GetRefreshInterval(instanceId)
	if err != nil {
		return err
	}

	fenceEnd := time.Now()
	fenceBegin := fenceEnd.Add(-2 * time.Duration(interval) * time.Second)

	// Clear old data for idempotency
	if err := m.clear(instanceId, fenceEnd, fenceBegin); err != nil {
		return err
	}

	// Sync statement data
	if err := m.syncHistory(instanceId, fenceBegin, fenceEnd); err != nil {
		return err
	}
	if err := m.syncCurrent(instanceId); err != nil {
		return err
	}

	return nil
}

func (m *statement) clear(instanceId string, start, end time.Time) error {
	return m.db.Delete(
		&Statement{}, 
		"INSTANCE_ID = ? AND SUMMARY_BEGIN_TIME >= ? AND SUMMARY_END_TIME <= ?", 
		instanceId, 
		start, 
		end,
	).Error()
}

func (m *statement) syncHistory(instanceId string, start, end time.Time) error {
	conns, err := m.peekAllConnection(instanceId)
	if err != nil {
		return err
	}
	defer func() { 
		for _, conn := range conns {
			conn.Close()
		}
	}()

	for _, conn := range conns {
		stmts := []*Statement{}

		// Load statements from TiDB
		if err := conn.Table("events_statements_summary_by_digest_history").Where(
			&Statement{}, 
			"SUMMARY_BEGIN_TIME >= ? AND SUMMARY_END_TIME <= ?", 
			start, 
			end,
		).Find(&stmts).Error(); err != nil {
			return err
		}

		for _, stmt := range stmts {
			stmt.InstanceId = instanceId
			stmt.Node = conn.instance
		}

		if err := m.db.Save(stmts).Error(); err != nil {
			return err
		}
	}

	return nil
}

func (m *statement) syncCurrent(instanceId string) error {
	conns, err := m.peekAllConnection(instanceId)
	if err != nil {
		return err
	}
	defer func() { 
		for _, conn := range conns {
			conn.Close()
		}
	}()

	for _, conn := range conns {
		stmts := []*Statement{}

		// Load statements from TiDB
		if err := conn.Table("events_statements_summary_by_digest").Find(&stmts).Error(); err != nil {
			return err
		}

		for _, stmt := range stmts {
			stmt.InstanceId = instanceId
			stmt.Node = conn.instance
		}

		if err := m.db.Save(stmts).Error(); err != nil {
			return err
		}
	}

	return nil
}