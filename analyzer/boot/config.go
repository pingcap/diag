package boot

import (
	"path"
)

// The config is use in task executing by other tasks
type Config struct {
	// The InspectionId indicate which inspection for analyze
	InspectionId string

	// Home is the work directory of foresight, where should be
	// bin, inspection, remote-log etc.
	Home string

	// This indicate the binary directory of home directory, where
	// lays the tools used by foresight
	Bin string

	// Simply path.Join(Home, InspectionId)
	Src string

	// Result of path.Join(Home, "remote-log")
	Logs string

	// Result of path.Join(Home, "profile")
	Profile string
}

func newConfig(inspectionId, home string) *Config {
	return &Config{
		inspectionId,
		home,
		path.Join(home, "bin"),
		path.Join(home, "inspection", inspectionId),
		path.Join(home, "remote-log", inspectionId),
		path.Join(home, "profile", inspectionId),
	}
}
