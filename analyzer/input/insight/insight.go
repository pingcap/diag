package insight

type Insight []*InsightInfo

type InsightInfo struct {
	// The host which owns this information
	NodeIp string

	// Info about tidb
	Meta struct {
		// tidb versions
		Tidb []struct {
			Version string `json:"release_version"`
		} `json:"tidb"`
		// tikv versions
		Tikv []struct {
			Version string `json:"release_version"`
		} `json:"tikv"`
		// pd versions
		Pd []struct {
			Version string `json:"release_version"`
		} `json:"pd"`
	} `json:"meta"`

	// Info about system itself
	Sysinfo struct {
		Os struct {
			// The name of os, eg. CentOS Linux 7 (Core)
			Name string `json:"name"`
		} `json:"os"`
		Kernel struct {
			// The release of os kernel, eg. 3.10.0-957.el7.x86_64
			Release string `json:"release"`
		}
		Cpu struct {
			// Model of cpu, eg. Intel(R) Xeon(R) CPU E5-2630 v4 @ 2.20GHz
			Model string `json:"model"`
		} `json:"cpu"`
		Memory struct {
			// Type of memory, eg. DDR4
			Type string `json:"type"`
			// Speed of memory
			Speed int `json:"speed"`
			// Size of memory
			Size int `json:"size"`
		} `json:"memory"`
		// Disk
		Storage []struct {
			// eg. nvme0n1
			Name string `json:"name"`
			// eg. sd
			Driver string `json:"driver"`
		} `json:"storage"`
		Network []struct {
			// eg. eth0
			Name string `json:"name"`
			// The speed of network card
			Speed int `json:"speed"`
		} `json:"network"`
	} `json:"sysinfo"`

	Ntp struct {
		// Tpical value is "sync_ntp"
		Sync string `json:"sync"`
		// The offset between this server and ntp server
		Offset float64 `json:"offset"`
		// Status code of ntp
		Status string `json:"status"`
	} `json:"ntp"`

	Partitions []struct {
		// eg. sda
		Name   string `json:"name"`
		Subdev []struct {
			// eg. sda1
			Name string `json:"name"`
		} `json:"subdev"`
	} `json:"partitions"`

	BlockInfo struct {
		Disks struct {
			Name string `json:"name"`
			// can be "unknown", "hdd", "ssd", etc
			DriveType string `json:"drive_type"`
			// can be "scsi", "nvme", etc
			StorageController string `json:"storage_controller"`

			Partitions []struct {
				// eg. sda
				Name       string `json:"name"`
				MountPoint string `json:"mount_point"`
			} `json:"partitions"`
		} `json:"disks"`
	} `json:"block_info"`
}
