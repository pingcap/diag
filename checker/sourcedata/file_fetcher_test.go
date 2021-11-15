package sourcedata

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/pingcap/diag/checker/config"
	"github.com/pingcap/diag/checker/proto"
)

func TestFileFetcher_loadSlowPlanData(t *testing.T) {
	bs, err := os.ReadFile("../testdata/avg_process_time_by_plan.csv")
	if err != nil {
		t.Error(err)
	}
	ff, err := NewFileFetcher("")
	if err != nil {
		t.Error(err)
	}

	res, err := ff.loadSlowPlanData(bytes.NewReader(bs))
	if err != nil {
		t.Error(err)
	}
	expected := map[string][2]proto.ExecutionPlanInfo{
		"1ff0c8be4c65117d55692b2ec06cc4d28050d7525d1e2bc0b0d8ddb599b85b83": {
			{PlanDigest: "0d21511d59e5dfca614ad7fbdd4dec175ea4bdd5c69cd317f6ac1bda1127053c", MaxLastTime: 1634805324, AvgProcessTime: 53},
			{PlanDigest: "4292d7aba1afb57e7314e7a44078323db029fb219f84e340502e75186bb038b3", MaxLastTime: 1634805292, AvgProcessTime: 72},
		},
		"20fbc1588c39df832b2b51f17a125e1a528bdb828d45925b5000eb68375b2b58": {
			{PlanDigest: "b4ab32154568affda9822187b21101b8ff7af5319d442c514df4062efc9a4e06", MaxLastTime: 1634803008, AvgProcessTime: 206},
			{PlanDigest: "53f7fc047d5abad21f32720e02428492b434819bd6a7b937b746b4868df30495", MaxLastTime: 1634804722, AvgProcessTime: 228},
		},
		"eaf0fbdeb196f9967b2ebeaee2e03de824ca1cde78aa386dc3fe2c1a3bccff18": {
			{PlanDigest: "d463f92d39fc41e22f2c40fa92c32dad426a0a3451b55f3607a6b6fff1ea8d1f", MaxLastTime: 1634794638, AvgProcessTime: 198},
			{PlanDigest: "73bea99e28eb2f175c0ce208b86e0964c482f9a2f28d5af2f09bf36e0ae59ca5", MaxLastTime: 1634795847, AvgProcessTime: 236},
		},
	}
	if fmt.Sprint(res) != fmt.Sprint(expected) {
		t.Error("result is not expected")
	}
}

func TestFileFetcher_loadDigest(t *testing.T) {
	input := `Digest,Plan_Digest
20fbc1588c39df832b2b51f17a125e1a528bdb828d45925b5000eb68375b2b58,53f7fc047d5abad21f32720e02428492b434819bd6a7b937b746b4868df30495
eaf0fbdeb196f9967b2ebeaee2e03de824ca1cde78aa386dc3fe2c1a3bccff18,1075995a5eff7d924e2f1f1fc59d564762d70df43e88b5f0c2642418cb8b89ef
eaf0fbdeb196f9967b2ebeaee2e03de824ca1cde78aa386dc3fe2c1a3bccff18,0d53fbed643f0585f6b3a621143ba57a553fd280364cc619dc7dc13050bc8739`
	reader := bytes.NewBufferString(input)
	ff, err := NewFileFetcher("")
	if err != nil {
		t.Error(err)
	}

	res, err := ff.loadDigest(reader)
	if err != nil {
		t.Error(err)
	}
	if len(res) != 3 {
		t.Error("result is not expected")
	}
}

func TestFileFetcher_loadSysVariables(t *testing.T) {
	input := `VARIABLE_NAME,VARIABLE_VALUE
bootstrapped,True
tidb_server_version,49
system_tz,Asia/Shanghai
new_collation_enabled,False
tikv_gc_leader_uuid,5f27c4758c00011
tikv_gc_leader_desc,"host:test-ecom-tidb-1, pid:1, start at 2021-10-21 02:58:38.80615727 +0800 CST m=+0.12892006"
tikv_gc_leader_lease,20211110-10:55:38 +0800
tikv_gc_enable,true
tikv_gc_run_interval,10m0s
tikv_gc_life_time,10m0s
tikv_gc_last_run_time,20211110-10:47:38 +0800
tikv_gc_safe_point,20211110-10:37:38 +0800
tikv_gc_auto_concurrency,true
tikv_gc_mode,distribute
`
	reader := bytes.NewBufferString(input)
	ff, err := NewFileFetcher("")
	if err != nil {
		t.Error(err)
	}

	res, err := ff.loadSysVariables(reader)
	if err != nil {
		t.Error(err)
	}
	if len(res) != 14 {
		t.Error("result is not expected")
	}
	if res["tikv_gc_life_time"] != "10m0s" {
		t.Error("result is not expected")
	}
}

func TestFileFetcher_FetchData(t *testing.T) {
	fetch, err := NewFileFetcher("../testdata", WithCheckFlag(ConfigFlag))
	if err != nil {
		t.Error(err)
	}
	ruleSpec, err := config.LoadBetaRuleSpec()
	if err != nil {
		t.Error()
	}

	data, rSet, err := fetch.FetchData(ruleSpec)
	if err != nil {
		t.Error(err)
	}
	if len(data.NodesData) == 0 {
		t.Error("fetch empty NodeData")
	}
	if len(rSet) == 0 {
		t.Error("fetch empty rule set")
	}
}
