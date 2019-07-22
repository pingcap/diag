# tidb-foresight

## 目录结构

- web: 前端 ui 界面
- api: api server, 对前端提供 restful 接口
- pioneer: 集群连通性检测工具，接收 invertory.ini，检测 api 所在机器和 invertory.init 里的所有机器是否正常连通
- collector: 诊断元信息收集器，通过 ansible 和应用层接口到集群负载机器上收集用于诊断的信息，供后续分析
- analyzer: 分析器，从诊断元信息中分析出可用于前端展示的信息，并插入数据库
