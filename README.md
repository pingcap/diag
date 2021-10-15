# diag

## 目录结构

- checker: 对 collector 的数据进行检查，并给出潜在的问题列表与调整建议
- collector: 诊断元信息收集器，通过 SSH 和应用层接口到集群负载机器上收集用于诊断的信息，供后续分析
- scraper: 收集日志和配置文件时的辅助工具，用于在集群负载机器上筛选文件
- cmd: 命令行工具 cli 入口
- pkg: 公共包

## 开发指南

- 安装好所有依赖
    - tiup-cluster

## Makefile

* `make check`：执行静态代码检查
* `make test`：运行单元测试
* `make build`：编译所有内容

