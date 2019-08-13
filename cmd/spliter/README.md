# tidb-foresight log spliter
## 用法
将源目录中指定时间段内的日志复制到目标目录中

```
spliter --src={source_dir} --dst={target_dir} --begin={begin_time_in_log} --end={end_time_in_log}
```

- src: 日志来源目录
- dst: 日志目标目录，可以不存在
- begin: 时间范围的开始, 只有这个时间之后打印的日志才被复制，格式RFC3339: 2019-07-22T19:49:25+08:00
- end: 时间范围的结束，只有这个时间之前的日志才被复制，格式同begin
