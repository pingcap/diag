package tikv

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, checkGrpc())
	tasks = append(tasks, checkScheduler())
	tasks = append(tasks, checkStorage())
	tasks = append(tasks, checkCoprocessor())
	tasks = append(tasks, checkRaftstore())
	tasks = append(tasks, checkRocksDBRaft())
	tasks = append(tasks, checkRocksDBKV())

	return tasks
}
