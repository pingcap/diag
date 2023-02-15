module github.com/pingcap/diag

go 1.16

require (
	github.com/AstroProfundis/sysinfo v0.0.0-20220902033416-231991f6df3c
	github.com/AstroProfundis/tabby v1.1.1 // indirect
	github.com/BurntSushi/toml v1.2.1
	github.com/Masterminds/semver v1.5.0
	github.com/ScaleFT/sshkeys v1.2.0 // indirect
	github.com/bilibili/gengine v1.5.7
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/fatih/color v1.14.1
	github.com/fatih/structs v1.1.0
	github.com/gin-gonic/gin v1.8.2
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/strfmt v0.21.3
	github.com/go-openapi/swag v0.22.3
	github.com/go-sql-driver/mysql v1.7.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/influxdb v1.11.0
	github.com/joho/sqltocsv v0.0.0-20210428211105-a6d6801d59df
	github.com/joomcode/errorx v1.1.0
	github.com/json-iterator/go v1.1.12
	github.com/kataras/tablewriter v0.0.0-20180708051242-e063d29b7c23 // indirect
	github.com/klauspost/compress v1.15.15
	github.com/lensesio/tableprinter v0.0.0-20201125135848-89e81fc956e7
	github.com/lorenzosaino/go-sysctl v0.3.1
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/onsi/gomega v1.26.0
	github.com/otiai10/copy v1.9.0 // indirect
	github.com/pingcap/check v0.0.0-20211026125417-57bd13f7b5f0
	github.com/pingcap/errors v0.11.5-0.20210425183316-da1aaba5fb63
	github.com/pingcap/kvproto v0.0.0-20230213063737-f1dee547f028 // indirect
	github.com/pingcap/log v1.1.0
	github.com/pingcap/tidb-operator/pkg/apis v1.4.1
	github.com/pingcap/tidb/parser v0.0.0-20230202053355-337af61d9521
	github.com/pingcap/tiup v1.11.1-0.20230215071401-de5f71bb5e98
	github.com/pkg/errors v0.9.1
	github.com/prometheus/common v0.39.0
	github.com/prometheus/prom2json v1.3.2 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.6.1
	github.com/stretchr/testify v1.8.1
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/vishvananda/netlink v0.0.0-20210530105856-14e832ae1e8f
	go.etcd.io/etcd/api/v3 v3.5.7
	go.etcd.io/etcd/client/pkg/v3 v3.5.7
	go.etcd.io/etcd/client/v3 v3.5.7
	go.uber.org/multierr v1.9.0 // indirect
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.6.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.5.0
	google.golang.org/genproto v0.0.0-20230209215440-0dfe4f8abfcc // indirect
	google.golang.org/grpc v1.53.0 // indirect
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver v0.22.4 // indirect
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.90.0
	sigs.k8s.io/yaml v1.3.0 // indirect
)

replace (
	github.com/appleboy/easyssh-proxy => github.com/AstroProfundis/easyssh-proxy v1.3.10-0.20211209071554-9910ebdf514e
	k8s.io/kube-openapi => k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
)
