package engine

import (
	"reflect"
	"testing"

	"github.com/pingcap/diag/checker/proto"
	"github.com/pingcap/diag/checker/render"
)

func TestWrapper_PackageResult(t *testing.T) {
	type fields struct {
		SourceData     *proto.SourceDataV2
		Render         *render.ResultWrapper
		RuleResult     map[string]proto.PrintTemplate
		RuleSet        map[string]*proto.Rule
		computeUnitSet map[string]*ComputeUnit
	}
	type args struct {
		hd        *proto.HandleData
		resultset map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "nil",
			fields: fields{
				SourceData:     nil,
				Render:         nil,
				RuleSet:        make(map[string]*proto.Rule),
				RuleResult:     nil,
				computeUnitSet: make(map[string]*ComputeUnit),
			},
			args: args{
				hd: &proto.HandleData{
					UqiTag:  "xxx_xx:xx",
					Data:    nil,
					IsValid: false,
				},
				resultset: map[string]interface{}{"testrule": true},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := NewWrapper(nil, nil, nil)
			w.RuleSet = make(map[string]*proto.Rule)
			w.RuleSet["testrule"] = &proto.Rule{Name: "testrule", CheckType: "config"}
			if err := w.PackageResult(tt.args.hd, tt.args.resultset); (err != nil) != tt.wantErr {
				t.Errorf("Wrapper.PackageResult() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(w.RuleResult) != 1 {
				t.Errorf("Wrapper.PackageResult() RuleResult, wantErr %v", w.RuleResult)
			}
		})
	}
}

func TestWrapper_GetDataSet(t *testing.T) {
	type fields struct {
		SourceData     *proto.SourceDataV2
		Render         *render.ResultWrapper
		RuleResult     map[string]proto.PrintTemplate
		RuleSet        map[string]*proto.Rule
		computeUnitSet map[string]*ComputeUnit
	}
	type args struct {
		namestruct string
		sd         *proto.SourceDataV2
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*proto.HandleData
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "getDataTest",
			fields: fields{
				SourceData:     NewMockSourceData(),
				RuleResult:     nil,
				RuleSet:        nil,
				computeUnitSet: nil,
			},
			args: args{
				sd:         NewMockSourceData(),
				namestruct: "TidbConfig",
			},
			want: []*proto.HandleData{
				{
					UqiTag: "TidbConfig_xxx,xxx234:1111",
					Data: []proto.Data{
						&proto.TidbConfigData{TidbConfig: &proto.TidbConfig{}, Port: 1111, Host: "xxx,xxx234"},
					},
					IsValid: true,
				},
				{
					UqiTag: "TidbConfig_xxx,xxx145:2222",
					Data: []proto.Data{
						&proto.TidbConfigData{TidbConfig: &proto.TidbConfig{}, Port: 2222, Host: "xxx,xxx145"},
					},
					IsValid: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wrapper{
				SourceData:     tt.fields.SourceData,
				Render:         tt.fields.Render,
				RuleResult:     tt.fields.RuleResult,
				RuleSet:        tt.fields.RuleSet,
				computeUnitSet: tt.fields.computeUnitSet,
			}
			got, err := w.GetDataSet(tt.args.namestruct)
			if (err != nil) != tt.wantErr {
				t.Errorf("Wrapper.GetDataSet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Wrapper.GetDataSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func NewMockSourceData() *proto.SourceDataV2 {
	sd := &proto.SourceDataV2{
		ClusterInfo: nil,
		TidbVersion: "v5.1",
		NodesData: map[string][]proto.Config{
			"PdConfig":   make([]proto.Config, 0),
			"TidbConfig": make([]proto.Config, 0),
		},
	}
	sd.NodesData["PdConfig"] = append(sd.NodesData["PdConfig"], &proto.PdConfigData{PdConfig: &proto.PdConfig{}, Port: 1234, Host: "xxx.xxx"})
	sd.NodesData["PdConfig"] = append(sd.NodesData["PdConfig"], &proto.PdConfigData{PdConfig: &proto.PdConfig{}, Port: 45345, Host: "xxx.xxxsdsd"})
	sd.NodesData["PdConfig"] = append(sd.NodesData["PdConfig"], &proto.PdConfigData{PdConfig: &proto.PdConfig{}, Port: 999, Host: "xxx.sdfaxx"})
	sd.NodesData["TidbConfig"] = append(sd.NodesData["TidbConfig"], &proto.TidbConfigData{TidbConfig: &proto.TidbConfig{}, Port: 1111, Host: "xxx,xxx234"})
	sd.NodesData["TidbConfig"] = append(sd.NodesData["TidbConfig"], &proto.TidbConfigData{TidbConfig: &proto.TidbConfig{}, Port: 2222, Host: "xxx,xxx145"})
	return sd
}

func TestWrapper_CrossData(t *testing.T) {
	w := NewMockSourceData()
	pdconf := w.NodesData["PdConfig"]
	tidbconf := w.NodesData["TidbConfig"]
	pdd := make([]proto.Data, 0)
	tdd := make([]proto.Data, 0)
	for _, d := range pdconf {
		pdd = append(pdd, d)
	}
	for _, d := range tidbconf {
		tdd = append(tdd, d)
	}
	type fields struct {
		SourceData     *proto.SourceDataV2
		Render         *render.ResultWrapper
		RuleResult     map[string]proto.PrintTemplate
		RuleSet        map[string]*proto.Rule
		computeUnitSet map[string]*ComputeUnit
	}
	type args struct {
		oriData [][]proto.Data
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   [][]proto.Data
	}{
		// TODO: Add test cases.
		{
			name:   "test",
			fields: fields{},
			args: args{
				oriData: [][]proto.Data{pdd, tdd},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Wrapper{
				SourceData:     tt.fields.SourceData,
				Render:         tt.fields.Render,
				RuleResult:     tt.fields.RuleResult,
				RuleSet:        tt.fields.RuleSet,
				computeUnitSet: tt.fields.computeUnitSet,
			}
			if got := w.CrossData(tt.args.oriData); len(got) != 6 && len(got) != 2 {
				t.Error(len(got), len(got[0]))
			}
		})
	}
}
