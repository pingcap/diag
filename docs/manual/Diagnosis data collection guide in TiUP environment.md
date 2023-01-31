# Diagnosis data collection guide in TiUP environment

## Summary
Diag is used to collect cluster diagnostic data, upload diagnostic data to the Clinic Server, and perform a quick health check locally on your cluster. For a full list of diagnostic data that can be collected by Diag, see [PingCAP Clinic Diagnostic Data](https://docs.pingcap.com/tidb/stable/clinic-data-instruction-for-tiup).
This doc describe different options when collecting data for a TiUP deployed cluster.

## Basic command
Collect the diagnostic data from 2 hours ago to current time, you can use the following command without any flags.
 
```bash
tiup diag collect ${cluster-name}
```
In this way, the Metrics,Logs,system info and config values will be collected. System variables data (db_vars), performance data (perf) and debug data (debug) will not be collected by default.

## Collect data for a special time range
You can define the time range with '-f' and '-t' parameters. For example, to collect the diagnostic data from 4 hours ago to 2 hours ago based on the current time, run the following command:

```bash
tiup diag collect ${cluster-name} -f="-4h" -t="-2h"
```
To specify an absolut time range, run the following command:

```bash
tiup diag collect ${cluster-name} -f="2023-01-31T10:04:26+08:00" -t="2023-01-31T13:04:26+08:00"
```
If you do not specify the time zone information in this parameter, such as +0800, the time zone is UTC by default. 

## Collect selected type of diagnosis data
You can use '--include' and '--exclude' parameters to filter the diagnosis data by type. All the available types are list in [PingCAP Clinic Diagnostic Data](https://docs.pingcap.com/tidb/stable/clinic-data-instruction-for-tiup). If you do not specify these two parameters, the default collection type includes 'system,config,monitor,log.std,log.slow'. 

If you only need metrics, run the following command:

```bash
tiup diag collect ${cluster-name} --include="monitor.metrics" -f="-4h" -t="-2h"
```

If you do not need slowquery log and system info, run the following command:

```bash
tiup diag collect ${cluster-name} --exclude="log.slow,system" -f="-4h" -t="-2h"
```

## Collect diagnosis data for selected node
If you only need the log data for tidb servers, run the following command:

```bash
tiup diag collect ${cluster-name} --include="log" -R="tidb"
```

If you only need the config data for node '127.0.0.1', run the following command:

```bash
tiup diag collect ${cluster-name} --include="config" -N="127.0.0.1"
``` 

> ** tip **
> Metrics and alerts collection can not be filtered by '-R' and '-N' parameters. When you have '--include=monitor' in your collect command, metrics and alerts for all nodes will be collected. 
> ** tip **
 
## Collect specific metrics 
Metrics can be filtered by '--metricsfilter' command，it can filter the metrics by metrics name. 
If you want to collect metrics whose name start with 'tidb', run the following command:

```bash
tiup diag collect ${cluster-name} --include="monitor" --metricsfilter="tidb"
``` 

If you want to collect all PD related metrics and rebuild PD dashboard in Clinic Server, run the following command:

```bash
tiup diag collect ${cluster-name} --include="monitor" --metricsfilter="pd,process,go,grpc,etcd"
```

## Adapt metrics collection limit
The metrics collection will use the Memory of Prometheus server. When the Prometheus resource is limited ,we need limit the collection request size to avoid OOM of Prometheuse server. 
To limit the request , run the following command:

```bash
tiup diag collect ${cluster-name} --metricslimit="1000"
```

