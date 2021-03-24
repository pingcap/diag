PROJECT=tidb-foresight


.PHONY: all collector spliter syncer fmt check_prefix build default

default: all

GOPATH ?= $(shell go env GOPATH)
# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
ifneq ($(MAKECMDGOALS),install)
	$(error Please set the environment variable GOPATH before running `make`)
else
	GOPATH := toinstall
endif

endif

CURDIR := $(shell pwd)
path_to_add := $(addsuffix /bin,$(subst :,/bin:,$(GOPATH))):$(PWD)/tools/bin
export PATH := $(path_to_add):$(PATH)

GO        := GO111MODULE=on go
GOBUILD   := CGO_ENABLED=1 $(GO) build $(BUILD_FLAG) -tags codes
GOTEST    := CGO_ENABLED=1 $(GO) test -p 4
OVERALLS  := CGO_ENABLED=1 GO111MODULE=on overalls

ARCH      := "`uname -s`"
LINUX     := "Linux"
MAC       := "Darwin"
PACKAGE_LIST  := go list ./...| grep -vE "cmd" | grep -vE "test"
PACKAGES  := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/$(PROJECT)/||'
FILES     := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")

FAILPOINT_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl enable)
FAILPOINT_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl disable)

FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

LDFLAGS += -X "github.com/pingcap/tidb-foresight/version.ReleaseVersion=$(shell git describe --tags --dirty --always)"
LDFLAGS += -X "github.com/pingcap/tidb-foresight/version.BuildTS=$(shell date -u '+%Y-%m-%d %I:%M:%S')"
LDFLAGS += -X "github.com/pingcap/tidb-foresight/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "github.com/pingcap/tidb-foresight/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"

CHECK_LDFLAGS += $(LDFLAGS)

INFLUXDB = influxd
PROMETHEUS = prometheus
PERL_SCRIPTS := flamegraph.pl fold-tikv-threads-perf.pl stackcollapse-perf.pl
NEEDS_INSTALL = $(INFLUXDB) $(PROMETHEUS) $(PERL_SCRIPTS)

DOWNLOAD_PREFIX = http://fileserver.pingcap.net/download/foresight/

ifeq ($(foresight_port),)
foresight_port=9527
endif
ifeq ($(influxd_port),)
influxd_port=9528
endif
ifeq ($(prometheus_port),)
prometheus_port=9529
endif

# set the default user tidb
ifeq ($(user), )
user=tidb
endif

all: collector spliter syncer

build:
	$(GOBUILD)

fmt:
	@echo "gofmt (simplify)"
	@gofmt -s -l -w $(FILES) 2>&1 | $(FAIL_ON_STDOUT)

RACE_FLAG =
ifeq ("$(WITH_RACE)", "1")
	RACE_FLAG = -race
	GOBUILD   = GOPATH=$(GOPATH) CGO_ENABLED=1 $(GO) build
endif

CHECK_FLAG =
ifeq ("$(WITH_CHECK)", "1")
	CHECK_FLAG = $(TEST_LDFLAGS)
endif

collector:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/collector cmd/collector/*.go

spliter:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/spliter cmd/spliter/*.go

syncer:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/syncer cmd/syncer/*.go

check-%:
	@ if [ "${${*}}" = "" ]; then \
		echo "variable $* not set"; \
		exit 1; \
	fi
