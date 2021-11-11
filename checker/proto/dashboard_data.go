package proto

type DashboardData struct {
	ExecutionPlanInfoList map[string][2]ExecutionPlanInfo // e.g {"xsdfasdf22sdf": {{min_execution_info}, {max_execution_info}}}
	OldVersionProcesskey  struct {
		GcLifeTime int // s
		Count      int
	}
	TombStoneStatistics struct {
		Count int
	}
}

type ExecutionPlanInfo struct {
	PlanDigest     string
	MaxLastTime    int64
	AvgProcessTime int64 // ms
}

func (d *DashboardData) ActingName() string {
	return "dashboard"
}
