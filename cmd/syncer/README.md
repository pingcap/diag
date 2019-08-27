# tidb-foresight log syncer
## 用法
监听 topology 目录下的 json 文件，解析出日志存放位置等信息，调用 rsync 将这些日志同步到中控机的指定目录中

```
syncer --topo={topo_dir} --target={target_dir} --interval={interval} --bwlimit={bandwidth_limit} --threshold=threshold}
```

- topo: 需要监听的 topology 目录
- target: 指定存放的位置
- interval: 同步任务的时间间隔（second）
- bwlimit: 限制每个 rsync 进程带宽速度（byte/s）
- threshold: 当日志量达到这个值时执行清理
