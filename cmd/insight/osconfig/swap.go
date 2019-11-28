package osconfig

const swapFilePath = "/proc/swaps"

func CollectSwaps() string {
	return cater(swapFilePath)
}