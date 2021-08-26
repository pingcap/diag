module github.com/pingcap/diag

go 1.16

require (
	github.com/alecthomas/repr v0.0.0-20181024024818-d37bc2a10ba1 // indirect
	github.com/fatih/color v1.12.0
	github.com/fatih/structs v1.1.0
	github.com/google/uuid v1.2.0
	github.com/influxdata/influxdb v1.9.3
	github.com/joomcode/errorx v1.0.3
	github.com/json-iterator/go v1.1.11
	github.com/klauspost/compress v1.13.4
	github.com/pingcap/check v0.0.0-20200212061837-5e12011dc712
	github.com/pingcap/errors v0.11.5-0.20201126102027-b0a155152ca3
	github.com/pingcap/log v0.0.0-20201112100606-8f1e84a3abc8 // indirect
	github.com/pingcap/tidb-insight v0.4.0-dev.1.0.20210812035908-19d4c37ba1b9 // indirect
	github.com/pingcap/tidb-insight/collector v0.0.0-20210812035908-19d4c37ba1b9 // indirect
	github.com/pingcap/tiup v1.6.0-dev.0.20210819033350-8f4dc5dd94c8
	github.com/prometheus/common v0.29.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	go.uber.org/zap v1.17.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/appleboy/easyssh-proxy => github.com/AstroProfundis/easyssh-proxy v1.3.10-0.20210615044136-d52fc631316d
