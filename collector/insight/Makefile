### Makefile for collector
GOPKG=github.com/pingcap/diag/collector/insight
ifeq "$(GOBIN)" ""
  GO    := go
else
  GO    := $(GOBIN)
endif
GOOS    := $(if $(GOOS),$(GOOS),"")
GOARCH  := $(if $(GOARCH),$(GOARCH),"")
GOENV   := GO111MODULE=on CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH)
GOBUILD := $(GOENV) $(GO) build $(BUILD_FLAGS)

COMMIT  := $(shell git describe --tags --always --dirty)
BRANCH  := $(shell git rev-parse --abbrev-ref HEAD)

LDFLAGS += -X "$(GOPKG)/sysinfo.GitBranch=$(BRANCH)"
LDFLAGS += -X "$(GOPKG)/sysinfo.GitCommit=$(COMMIT)"

default: build

build: insight

insight:
	$(GOBUILD) -ldflags '-s -w -extldflags "-static" $(LDFLAGS)' \
		-o ../../bin/insight \
		collector.go

test:
	$(GO) test ./...

check:
	@echo "no check needed, skip"
