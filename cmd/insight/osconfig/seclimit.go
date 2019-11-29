package osconfig

const limitFilePath = "/etc/security/limits.conf"

func CollectSecLimits() string {
	return cater(limitFilePath)
}
