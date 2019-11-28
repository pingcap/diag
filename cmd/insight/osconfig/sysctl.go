package main

import (
	"fmt"
	sysctl "github.com/lorenzosaino/go-sysctl"
	"github.com/pingcap/tidb-foresight/utils/debug_printer"
)

func main()  {
	msg, err := sysctl.GetAll()
	if err != nil {
		panic(err)
	}
	fmt.Println(debug_printer.FormatJson(msg))
}