package osconfig

import (
	sysctl "github.com/lorenzosaino/go-sysctl"
	"github.com/pingcap/tidb-foresight/utils/debug_printer"
)

// If the machine is not an Linux system, the CollectSysctl will panic rather than raise an error.
func CollectSysctl() string {
	msg, err := sysctl.GetAll()
	if err != nil {
		panic(err)
	}
	return debug_printer.FormatJson(msg)
}
