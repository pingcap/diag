package software

import (
	"github.com/pingcap/tidb-foresight/analyzer/boot"
	"github.com/pingcap/tidb-foresight/analyzer/input/insight"
)

type saveOSConfigTask struct{}

func SaveOSConfig() *saveOSConfigTask {
	return &saveOSConfigTask{}
}

// Save each component's config to database
func (t *saveOSConfigTask) Run(m *boot.Model, c *boot.Config, insight *insight.Insight) {
	// num_fds
	panic("implement me")
}
