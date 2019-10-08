PROJECT=tidb
GOPATH ?= $(shell go env GOPATH)

# Ensure GOPATH is set before running build process.
ifeq "$(GOPATH)" ""
  $(error Please set the environment variable GOPATH before running `make`)
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

# TODO: remove debug
.PHONY: all server analyzer spliter syncer install debug

default: all	

all: server analyzer spliter syncer
	eval './download.py $(DOWNLOAD_PREFIX) $(NEEDS_INSTALL)'

# If prefix is now provided, please abort
# it will execute after all the target is already build
# Usage: make install prefix=/opt/tidb
# TODO: if we need a default prefix?
install: all
ifndef prefix
	$(error prefix is not set)
endif
	eval './install.py $(prefix) $(NEEDS_INSTALL)'

build:
	$(GOBUILD)

RACE_FLAG =
ifeq ("$(WITH_RACE)", "1")
	RACE_FLAG = -race
	GOBUILD   = GOPATH=$(GOPATH) CGO_ENABLED=1 $(GO) build
endif

CHECK_FLAG =
ifeq ("$(WITH_CHECK)", "1")
	CHECK_FLAG = $(TEST_LDFLAGS)
endif

server:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/tidb-foresight cmd/server/*.go

analyzer:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/analyzer cmd/analyzer/*.go

spliter:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/spliter cmd/spliter/*.go

syncer:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/syncer cmd/syncer/*.go
