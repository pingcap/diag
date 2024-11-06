package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricFilter(t *testing.T) {
	assert := require.New(t)

	list := []string{
		"tikv_xxx",
		"tidb_xxx",
		"ticdc_xxx",
		"node_xxx",
	}
	filter := []string{
		"tikv",
		"tidb",
		"node",
	}
	exclude := []string{
		"node",
	}

	assert.Equal(filterMetrics(list, filter, exclude), []string{"tikv_xxx", "tidb_xxx"})
	assert.Equal(filterMetrics(list, nil, exclude), []string{"tikv_xxx", "tidb_xxx", "ticdc_xxx"})
	assert.Equal(filterMetrics(list, filter, nil), []string{"tikv_xxx", "tidb_xxx", "node_xxx"})
	assert.Equal(filterMetrics(list, nil, nil), []string{"tikv_xxx", "tidb_xxx", "ticdc_xxx", "node_xxx"})
}
