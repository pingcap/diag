package emphasis

import (
	"github.com/pingcap/tidb-foresight/bootstrap"
	"github.com/pingcap/tidb-foresight/model"
)

type createEmphasisHandler struct {
	c *bootstrap.ForesightConfig
	m model.Model
}

// TODO: 以后删掉。透，我怎么感觉这是样板工作，没有啥好用的 web framework 么
