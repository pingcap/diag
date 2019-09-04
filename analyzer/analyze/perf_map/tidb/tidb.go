package tidb

func Tasks() []interface{} {
	tasks := make([]interface{}, 0)

	tasks = append(tasks, checkConnection())
	tasks = append(tasks, checkHeapMemory())
	tasks = append(tasks, checkTokenLimit())
	tasks = append(tasks, checkParseDuration())
	tasks = append(tasks, checkCompileDuration())
	tasks = append(tasks, checkTransaction())
	tasks = append(tasks, checkTso())
	tasks = append(tasks, checkKV())

	return tasks
}
