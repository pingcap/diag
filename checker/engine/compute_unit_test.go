package engine

import (
	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/proto"
	"github.com/stretchr/testify/require"
	"testing"
)

// rule 105 should return ok if `performance.feedback-probability` is set to 0, otherwise it should return false.
func Test_Issue243(t *testing.T) {
	assert := require.New(t)
	// warning case
	{
		// construct config
		cfg := proto.NewTidbConfigData()
		cfg.Performance.FeedbackProbability = 0.5
		// construct rule
		rules, err := config.LoadBetaRuleSpec()
		assert.Nil(err)
		rs, err := rules.FilterOn(func(item config.RuleItem) (bool, error) {
			if item.ID == 105 {
				return true, nil
			}
			return false, nil
		})
		assert.Nil(err)

		// create compute unit\
		hd := proto.NewHandleData([]proto.Data{cfg})
		cu := NewComputeUnit(hd)
		for _, val := range rs {
			cu.Rules = append(cu.Rules, val)
		}
		// compute unit compute result, should return false
		result, err := cu.Compute()
		assert.Nil(err)
		for _, v := range result {
			assert.Equal(v, false)
		}
	}
	// ok case
	{
		// construct config
		cfg := proto.NewTidbConfigData()
		cfg.Performance.FeedbackProbability = 0
		// construct rule
		rules, err := config.LoadBetaRuleSpec()
		assert.Nil(err)
		rs, err := rules.FilterOn(func(item config.RuleItem) (bool, error) {
			if item.ID == 105 {
				return true, nil
			}
			return false, nil
		})
		assert.Nil(err)

		// create compute unit\
		hd := proto.NewHandleData([]proto.Data{cfg})
		cu := NewComputeUnit(hd)
		for _, val := range rs {
			cu.Rules = append(cu.Rules, val)
		}
		// compute unit compute result, should return true
		result, err := cu.Compute()
		assert.Nil(err)
		for _, v := range result {
			assert.Equal(v, true)
		}
	}

}
