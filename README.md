[![LICENSE](https://img.shields.io/github/license/pingcap/tidb.svg)](https://github.com/pingcap/diag/blob/master/LICENSE)
[![Language](https://img.shields.io/badge/Language-Go-blue.svg)](https://golang.org/)
[![Go Report Card](https://goreportcard.com/badge/github.com/pingcap/diag)](https://goreportcard.com/badge/github.com/pingcap/diag)
[![Coverage Status](https://codecov.io/gh/pingcap/diag/branch/master/graph/badge.svg)](https://codecov.io/gh/pingcap/diag/)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fpingcap%2Fdiag.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fpingcap%2Fdiag?ref=badge_shield)

# What is diag

`diag` is a tool to collect diagnostic data from TiDB clusters. It is mainly designed to be used for TiDB clusters deployed by `tiup-cluster` but could also be used for manually deployed ones and TiDB clusters on Kubernetes with limited features.

## CN2 实例 IP 脱敏版使用说明

本分支的完整中文文档位于 [docs/manual/CN2-IP-redaction.md](docs/manual/CN2-IP-redaction.md)。
文档包含 Linux 和 Mac 安装、采集、IP 脱敏、空间占用、打包上传及常见错误。
以下参数需要使用本分支构建的程序；官方 TiUP 安装的版本可能没有这些选项。

保留 IP 最后一段，前三段固定为 `10.0.0`，例如 `172.16.34.56 → 10.0.0.56`：

```bash
./diag collect my-cluster \
  --output ./diag-data \
  --redact-all-ips \
  --keep-ipv4-octets 1 \
  --redact-in-place
```

`my-cluster` 替换为实际集群名。上述命令替换本地采集目录中的内容，只保留一份脱敏数据。
服务器上的原始日志保持不变。处理仍需要当前文件和归档内部的临时空间。
不同 IP 的最后一段相同，或原地址已经是 `10.0.0.x` 时，程序会报错。
保留尾段属于部分隐藏，不能当作加密。

| 保存方式 | 参数 | 处理完成后保留的数据 |
| --- | --- | --- |
| 原地替换 | `--output ./diag-data --redact-in-place` | 一份脱敏数据，覆盖本地原始采集内容 |
| 保留副本 | `--output ./diag-original --redact-output ./diag-redacted` | 一份原始数据和一份脱敏数据 |

两种方式都需要选择脱敏模式，例如 `--redact-all-ips`。
`--redact-in-place` 与 `--redact-output` 互斥。
脱敏目录中的内容尚未加密；执行 `diag package` 后，才会额外生成加密的 `.diag` 文件。
本地 IP 对照表的导出和保管方式见完整文档，不能随诊断包上传。

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
