package sysinfo

// Kernel information.
type Kernel struct {
	Release      string `json:"release,omitempty"`
	Version      string `json:"version,omitempty"`
	Architecture string `json:"architecture,omitempty"`
}

func (si *SysInfo) getKernelInfo() {
	si.Kernel.Release = "unimplemented"
	si.Kernel.Version = "unimplemented"
	si.Kernel.Architecture = "darwin"
}
