# 诊断收集器
## 用法
```
python collector.py 
    --inspection-id={uuid} 
    --data-dir=/dir/to/store 
    --inventory=inventory.ini
    --topology=topology.json 
    --collect=basic,profile,dbinfo,slowlog:1h,metric:1h
```
## 元信息目录
```
+{inspection_id}
|-- collect.txt	                // --collect参数内容
|-- topology.json               // --topology参数指定的文件拷贝到这里
|--+ basic				        // 基础信息
|  |-- meta.json			    // 集群名称，创建时间，诊断时间
|  |-- {pd, tidb, tikv}.json	// active 进程信息 (来源prometheus)
|--+ insight				    // tidb-insight collector收集的机器信息
|  |--+ {ip}
|     |-- collector.json
|--+ profile
|  |--+ {ip}
|     |-- tikv_{cpu, heap}.data	// 在目标机执行perf record tikv收集到的数据
|     |-- {pd, tidb}_{heap, cpu}// 通过http接口下载到的golang的pprof数据
|--+ dbinfo				        // 数据库信息，通过tidb的http接口获得
|  |-- schema.json
|  |-- {db}.json
|  |--+ {db}
|     |-- {table}.json
|--+ resource				    // 集群资源信息，prometheus返回的结果
|  |-- {vcore, mem_total, mem_avail, cpu, load, network_inbound, network_outbound}.json
|  |-- {tcp_retrans_syn, tcp_retrans_slow_start, tcp_retrans_forward, io}.json
|-- alert.json				    // 当前的告警信息
|--+ slow_log
|  |--+ {ip}
|     |-- log
|--+ config				        // 集群组件配置
|  |--+ {ip}
|       |-- {tidb, tikv, pd}.toml
|--+ network				    // 网络质量
|  |--+ {ip}
|     |-- ping_{ip}			// ping其他机器的结果
|     |-- netstat
|     |-- ss
|--+ metric				        // 监控数据，来源prometheuse
|  |-- http_requests_total_1561445611_to_1561452811_15.0s.json
|  |-- ...
|--+ vars				        // TiDB运行时变量
|  |-- global.json			    // 全局变量
|  |-- {ip}.json				// 局部变量
|--+ demsg				        // demsg信息
|  |--+ {ip}
|     |-- log·
|--+ proc				        // proc目录下相关信息
   |-- {iostat, mpstat, vmstat, pidstat}

```
## 文件格式
待定