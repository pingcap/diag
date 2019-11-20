package resource

// The Resource represent the cluster resource usage in the metric time range
type Resource struct {
	AvgCPU    float64
	MaxCPU    float64
	AvgMem    float64
	MaxMem    float64
	AvgIoUtil float64
	MaxIoUtil float64
	AvgDisk   float64
	MaxDisk   float64

	// max ping duration.
	MaxDuration float64
	// minimum ping duration.
	MinDuration float64
	// average ping duration.
	AvgDuration float64
}
