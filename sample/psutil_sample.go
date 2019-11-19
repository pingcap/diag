package main

import (
	"encoding/json"
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/mem"
)

func PrintJsonData(dataStruct interface{}) {
	message, err := json.MarshalIndent(dataStruct, "    ", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", string(message))
}

func main() {
	v, _ := mem.VirtualMemory()

	// almost every return value is a struct
	fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)

	// convert to JSON. String() is also implemented
	fmt.Println(v)

	block, err := ghw.Block()
	if err != nil {
		fmt.Printf("Error getting block storage info: %v", err)
	}

	fmt.Printf("%v\n", block)

	for _, disk := range block.Disks {
		fmt.Println(disk)
		message, err := json.MarshalIndent(disk, "", "    ")
		if err != nil {
			panic(err)
		}
		fmt.Printf(" %v\n", string(message))
		for _, part := range disk.Partitions {
			fmt.Printf("  %v\n", part)
		}
	}
}
