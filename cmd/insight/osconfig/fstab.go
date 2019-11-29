package osconfig

import (
	fstab "github.com/d-tux/go-fstab"
	"github.com/pingcap/tidb-foresight/utils/debug_printer"
)

func CollectFstab() string {
	mount, err := fstab.ParseSystem()
	if err != nil {
		panic(err)
	}

	return debug_printer.FormatJson(mount)
}
