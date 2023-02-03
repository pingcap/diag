[![LICENSE](https://img.shields.io/github/license/pingcap/tidb.svg)](https://github.com/pingcap/diag/blob/master/LICENSE)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/pingcap/diag)](https://goreportcard.com/badge/github.com/pingcap/diag)
[![Coverage Status](https://codecov.io/gh/pingcap/diag/branch/master/graph/badge.svg)](https://codecov.io/gh/pingcap/diag/)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpingcap%2Fdiag.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpingcap%2Fdiag?ref=badge_shield)

# What is diag

`diag` is a tool to collect diagnostic data from TiDB clusters. It is mainly designed to be used for TiDB clusters deployed by `tiup-cluster` but could also be used for manually deployed ones and TiDB clusters on Kubernetes with limited features.

# Quick Start

Use [TiUP](https://github.com/pingcap/tiup) to install `diag`:

```
tiup install diag
```

Then use the `diag` command to collect data from a TiDB cluster:

```
tiup diag collect ${cluster-name} -f="-4h" -t="-2h"
```

Where `${cluster-name}` is the name of TiDB cluster deployed by `tiup-cluster`, and the `-f/-t` arguments means to collect the diagnostic data from 4 hours ago to 2 hours ago based on the current time.

## Online Clinic Service

The data collected from above procedures could be viewed locally for manual investigation or recording usage. There is also an online service that could parse these data and provide more analystic and diagnostic features.

To upload collected data to the PingCAP Clinic, please follow [this guide](https://docs.pingcap.com/tidb/v6.3/quick-start-with-clinic).

# Development

Some design and specification docs are in the `docs` directory.

## File Structures

- checker: a simple static checker used to parse collected data and list known issues
- collector: a tool to collect system information of a server
- scraper: a tool used to locate log and config files on remote server
- cmd: command line entries for all these tools
- pkg: shared packages

## Building

- Install [Go](https://golang.org/) version 1.16 or later
- Run `make build`

### The `Makefile`

Some extra entries are in `Makefile` and could be useful:

* `make check`：run static code check
* `make test`：run all unit tests
* `make build`：build all binaries

## Contributing to Diag

Contributions of code, tests, docs, and bug reports are welcome! To get started, take a look at our [open issues](https://github.com/pingcap/diag/issues).

See also the [Contribution Guide](https://github.com/pingcap/community/blob/master/contributors/README.md) in PingCAP's
[community](https://github.com/pingcap/community) repo.

## License

This project is licensed onder [Apache License Version 2.0](LICENSE).
