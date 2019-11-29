package osconfig

const fileCfg = "/etc/sysctl.conf"

// If the machine is not an Linux system, the CollectSysctl will panic rather than raise an error.
func CollectSysctl() string {
	return cater(fileCfg)
}
