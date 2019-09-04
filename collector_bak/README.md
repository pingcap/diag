# 诊断收集器

## 用法

```sh
./collect
    --inspection-id={uuid}
    --data-dir=/dir/to/store
    --inventory=inventory.ini
    --topology=topology.json
    --collect=basic,profile,dbinfo,log,metric
```

## 元信息目录

```sh
data/                                                       // 收集信息存储目录
└── {UUID}                                                  // inspection id 代表依次诊断
    ├── alert.json                                          // 集群报警信息，从 prometheus 采集
    ├── collect.json                                        // --collect 参数内容
    ├── config                                              // 配置文件
    │   ├── pd
    │   │   ├── {IP}:{PORT}
    │   │   │   └── pd.toml
    │   ├── tidb
    │   │   └── {IP}:{PORT}
    │   │       └── tidb.toml
    │   └── tikv
    │       ├── {IP}:{PORT}
    │       │   └── tikv.toml
    ├── dbinfo                                              // 数据库相关信息
    │   ├── information_schema.json
    │   ├── mysql.json
    │   ├── performance_schema.json
    │   └── test.json
    ├── dmesg                                               // dmesg 运行结果
    │   └── {IP}
    │       └── dmesg
    ├── insight                                             // tidb-insight collector 运行结果
    │   └── {IP}
    │       └── collector.json
    ├── meta.json                                           // 元信息，包括诊断的集群名称，开始时间，结束时间等
    ├── metric                                              // metric 信息，从 prometheus 采集
    │   ├── ALERTS_FOR_STATE_1563872437_to_1563876037_60s.json
    │   ├── etcd_cluster_version_1563872437_to_1563876037_60s.json
    │   ├── ...
    │   └── up_1563872437_to_1563876037_60s.json
    ├── net                                                 // 网络相关数据，目前只有 netstat -s 的运行结果
    │   └── {IP}
    │       └── netstat
    ├── proc                                                // iostat, mpstat, pidstat, vmstat 运行结果
    │   └── {IP}
    │       ├── iostat_1_60
    │       ├── mpstat_1_60
    │       ├── pidstat_1_60
    │       └── vmstat_1_60
    ├── profile                                             // go pprof 和 perf record 运行结果
    │   ├── pd
    │   │   ├── {IP}:{PORT}
    │   │   │   ├── allocs.pb.gz
    │   │   │   ├── block.pb.gz
    │   │   │   ├── cpu.pb.gz
    │   │   │   ├── mem.pb.gz
    │   │   │   ├── mutex.pb.gz
    │   │   │   ├── threadcreate.pb.gz
    │   │   │   └── trace.pb.gz
    │   ├── tidb
    │   │   └── {IP}:{PORT}
    │   │       └── ...
    │   └── tikv
    │       ├── {IP}:{PORT}
    │       │   └── perf.data
    └── topology.json                                        // 集群拓扑信息
```

## 文件格式

* HTTP 接口返回结果，按源格式写入文件
* 命令执行结果，按命令输出写入文件
* collect 程序生成数据，输出为 json 格式
