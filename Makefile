PROJECT=diag


.PHONY: all collector scraper k8s fmt vet lint check build default metricsconfig

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

GOOS    := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
GOARCH  := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))
GOENV   := GO111MODULE=on CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH)
GO      := $(GOENV) go
GOBUILD := $(GO) build $(BUILD_FLAGS)
GOTEST  := CGO_ENABLED=0 $(GO) test -p 4

DOCKER_REGISTRY ?= localhost:5000
DOCKER_REPO     ?= ${DOCKER_REGISTRY}/pingcap
IMAGE_TAG       ?= latest

PACKAGE_LIST        := go list ./...| grep -vE "cmd" | grep -vE "test"
PACKAGES            := $$($(PACKAGE_LIST))
PACKAGE_DIRECTORIES := $(PACKAGE_LIST) | sed 's|github.com/pingcap/$(PROJECT)/||'
FILES               := $$(find $$($(PACKAGE_DIRECTORIES)) -name "*.go")

FAILPOINT_ENABLE  := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl enable)
FAILPOINT_DISABLE := $$(find $$PWD/ -type d | grep -vE "(\.git|tools)" | xargs tools/bin/failpoint-ctl disable)

FAIL_ON_STDOUT := awk '{ print } END { if (NR > 0) { exit 1 } }'

LDFLAGS += -s -w
LDFLAGS += -X "github.com/pingcap/diag/version.ReleaseVersion=$(shell git describe --tags --dirty --always)"
LDFLAGS += -X "github.com/pingcap/diag/version.GitHash=$(shell git rev-parse HEAD)"
LDFLAGS += -X "github.com/pingcap/diag/version.GitBranch=$(shell git rev-parse --abbrev-ref HEAD)"

CHECK_LDFLAGS += $(LDFLAGS)

all: check build

build: diag insight scraper metricsconfig

clean:
	@rm -rf bin

diag:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/diag cmd/diag/*.go

insight:
	$(MAKE) -C collector/insight $(MAKECMDGOALS)

scraper:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o bin/scraper cmd/scraper/*.go

metricsconfig:
	cp metricsconfig/* bin/

k8s:
	$(GOBUILD) $(RACE_FLAG) -ldflags '$(LDFLAGS) $(CHECK_FLAG)' -o k8s/images/diag/bin/k8s-pod cmd/k8s/*.go

images: k8s
	docker build --tag "${DOCKER_REPO}/diag:${IMAGE_TAG}" -f k8s/images/diag/Dockerfile k8s/images/diag

test:
	$(GO) test ./...

check: fmt vet lint check-static

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

vet:
	$(GO) vet ./...

lint: tests/bin/revive
	@echo "linting"
	./tests/check/check-lint.sh
	@tests/bin/revive -formatter friendly \
		-exclude ./k8s/apis/... \
		-exclude ./api/types/... \
		-exclude ./collector/insight/kmsg/... \
		-config tests/check/revive.toml \
		$(FILES)

tests/bin/revive: tests/check/go.mod
	cd tests/check; \
	$(GO) build -o ../bin/revive github.com/mgechev/revive

tools/bin/golangci-lint:
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b ./tools/bin v1.41.1

check-static: tools/bin/golangci-lint
	GO111MODULE=on CGO_ENABLED=0 tools/bin/golangci-lint run -v --config .golangci.yml
