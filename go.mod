module github.com/pingcap/diag

go 1.16

require (
	github.com/BurntSushi/toml v0.4.1
	github.com/Masterminds/semver v1.4.2
	github.com/alecthomas/repr v0.0.0-20181024024818-d37bc2a10ba1 // indirect
	github.com/bilibili/gengine v1.5.7
	github.com/fatih/color v1.13.0
	github.com/fatih/structs v1.1.0
	github.com/google/uuid v1.3.0
	github.com/influxdata/influxdb v1.9.4
	github.com/joomcode/errorx v1.0.3
	github.com/json-iterator/go v1.1.12
	github.com/kataras/tablewriter v0.0.0-20180708051242-e063d29b7c23 // indirect
	github.com/klauspost/compress v1.13.6
	github.com/lensesio/tableprinter v0.0.0-20201125135848-89e81fc956e7
	github.com/onsi/gomega v1.16.0
	github.com/pingcap/check v0.0.0-20200212061837-5e12011dc712
	github.com/pingcap/errors v0.11.5-0.20201126102027-b0a155152ca3
	github.com/pingcap/log v0.0.0-20210906054005-afc726e70354
	github.com/pingcap/tidb-insight/collector v0.0.0-20210923072556-14ae4968ce78
	github.com/pingcap/tiup v1.7.0-dev.0.20211020035431-b38df71399f3
	github.com/prometheus/common v0.30.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

replace github.com/appleboy/easyssh-proxy => github.com/AstroProfundis/easyssh-proxy v1.3.10-0.20210615044136-d52fc631316d
