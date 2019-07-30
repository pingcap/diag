package main

// componentPattern is a list of special cases for the log file naming pattern of a component,
// By default, all components that begin with the component name will be matched.
// For example, the tikv component will match the log file of tikv*.log by default.
var componentPattern = map[string][]string{
	"prometheus": {"alertmanager"},
}
