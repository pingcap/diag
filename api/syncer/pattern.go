package syncer

// 默认会匹配所有以组件名开头的组件
// 比如 tikv 组件会默认匹配到 tikv*.log 的日志文件
// 下面列出的是特殊情况，包含不止一个文件名开头不同的日志文件
var componentPattern = map[string][]string{
	"prometheus": {"alertmanager"},
}
