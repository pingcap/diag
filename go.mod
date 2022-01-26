module github.com/pingcap/diag

go 1.16

require (
	github.com/AstroProfundis/sysinfo v0.0.0-20211201040748-b52c88acb418
	github.com/BurntSushi/toml v0.4.1
	github.com/Masterminds/semver v1.5.0
	github.com/alecthomas/repr v0.0.0-20181024024818-d37bc2a10ba1 // indirect
	github.com/bilibili/gengine v1.5.7
	github.com/fatih/color v1.13.0
	github.com/fatih/structs v1.1.0
	github.com/gin-gonic/gin v1.7.7
	github.com/go-openapi/errors v0.20.1 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/strfmt v0.21.1
	github.com/go-openapi/swag v0.19.15
	github.com/go-sql-driver/mysql v1.6.0
	github.com/google/go-cmp v0.5.6
	github.com/google/gofuzz v1.2.0
	github.com/google/uuid v1.3.0
	github.com/influxdata/influxdb v1.9.5
	github.com/joho/sqltocsv v0.0.0-20210428211105-a6d6801d59df
	github.com/joomcode/errorx v1.0.3
	github.com/json-iterator/go v1.1.12
	github.com/kataras/tablewriter v0.0.0-20180708051242-e063d29b7c23 // indirect
	github.com/klauspost/compress v1.13.6
	github.com/lensesio/tableprinter v0.0.0-20201125135848-89e81fc956e7
	github.com/lorenzosaino/go-sysctl v0.2.0
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/onsi/gomega v1.17.0
	github.com/pingcap/check v0.0.0-20211026125417-57bd13f7b5f0
	github.com/pingcap/errors v0.11.5-0.20201126102027-b0a155152ca3
	github.com/pingcap/log v0.0.0-20210906054005-afc726e70354
	github.com/pingcap/tidb-insight/collector v0.0.0-20211201041326-0f05f9ddcba2 // indirect
	github.com/pingcap/tiup v1.9.0-dev
	github.com/prometheus/common v0.32.1
	github.com/prometheus/prometheus v1.8.2-0.20210518124745-db7f0bcec27b
	github.com/shirou/gopsutil v3.21.10+incompatible
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink v0.0.0-20210530105856-14e832ae1e8f
	go.uber.org/zap v1.19.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211124211545-fe61309f8881
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211
	golang.org/x/text v0.3.7 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.30.0
	k8s.io/kube-openapi v0.0.0-20211109043538-20434351676c
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a
	sigs.k8s.io/yaml v1.3.0
)

replace github.com/appleboy/easyssh-proxy => github.com/AstroProfundis/easyssh-proxy v1.3.10-0.20211209071554-9910ebdf514e
