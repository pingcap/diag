package osconfig

type OsConfig struct {
	Fstab string `json:"fstab"`
	SecLimit string `json:"sec_limit"`
	Swap string `json:"swap"`
	Sysctl string `json:"sysctl"`
}

func Collect() OsConfig {
	return OsConfig{
		Fstab:    CollectFstab(),
		SecLimit: CollectSecLimits(),
		Swap:     CollectSwaps(),
		Sysctl:   CollectSysctl(),
	}
}

