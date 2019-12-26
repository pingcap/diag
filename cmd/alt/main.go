package main

import (
	"encoding/json"
	"fmt"
	"github.com/pingcap/tidb-foresight/analyzer/input/alert"
	"io/ioutil"
)

func main() {
	msg, err := ioutil.ReadFile("cmd/alt/alert_test.json")
	fmt.Printf(string(msg))

	if err != nil {
		panic(err)
	}
	var to alert.Alert

	err = json.Unmarshal(msg, &to)
	if err != nil {
		panic(err)
	}
	fmt.Printf("\n%+v\n", to)
	msgs, err := json.MarshalIndent(to, "", "\t")
	if err != nil {
		panic(err)
	}

	//fmt.Println(nmsl.Fuck["status"])
	fmt.Println(string(msgs))
}
