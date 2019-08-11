package dmesg

// Dmesg contains a list of ip addresses and their dmesg logs
type Dmesg []DmesgInfo

type DmesgInfo struct {
	Ip  string
	Log string
}
