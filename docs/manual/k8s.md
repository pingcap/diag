# Operation Manual for Operator cluster
For clusters deployed with [TiDB Operator](https://github.com/pingcap/tidb-operator), Diag needs to be deployed as a separate pod. This article describes how to use the `kubectl` command to create and deploy a Diag pod, and then continue data collection and quick inspection through API calls.

## Usage Scenarios
Through the Diag tool of the [Clinic](https://clinic.pingcap.com/) diagnostic service, you can easily and quickly obtain diagnostic data for basic diagnosis of the cluster:
- Collect diagnostic data using Diag
- Run simple local check with Diag

## Installation
This section details the steps to install the Diag diagnostic client.

### Prepare the environment
Before deploying Diag, please confirm the following software requirements:
- Kubernetes v1.12 or higher
- [TiDB Operator](https://docs.pingcap.com/zh/tidb-in-kubernetes/stable/tidb-operator-overview)
- [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
- [RBAC](https://kubernetes.io/docs/admin/authorization/rbac) enabled (optional)
- [Helm 3](https://helm.sh/)

### Installation Helm
Refer to [using Helm](https://clinic-docs.vercel.app/docs/getting-started/tidb-toolkit.md#%E4%BD%BF%E7%94%A8-helm) to install Helm and configure PingCAP official chart repository.

### Prepare the deployment permissions
The deployment user needs to have permission to create a Role and be able to create a cluster role with the following permissions:
```
  - apiGroups: [""]
    resources: ["pods", "services", "secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["pingcap.com"]
    resources: ["tidbclusters", "tidbmonitors"]
    verbs: ["get", "list"]
```

### Deploy the Clinic Diag Pod
1. Deploy Clinic Diag with the following helm command to download the latest Diag image from [Docker Hub](https://hub.docker.com/r/pingcap/diag)
```
# namespace： the same namespace as TiDB Operator 
helm install --namespace tidb-admin diag-collector pingcap/diag \      --set diag.clinicToken=${clinic_token }
```

2. After deployment, return as follows:
```
NAME: diag-collector
LAST DEPLOYED: Tue Mar 15 13:00:44 2022
NAMESPACE: tidb-adminSTATUS: deployed
REVISION: 1
NOTES:Make sure diag-collector components are running:
kubectl get pods --namespace tidb-admin -l app.kubernetes.io/instance=diag-collectorkubectl get svc --namespace tidb-admin -l app.kubernetes.io/name=diag-collector
```

### Check the running status of the Diag Pod
Use the following command to query the Diag status:
```
kubectl get pods --namespace tidb-admin -l app.kubernetes.io/instance=diag-collector
```

The output of the normal operation of the pod is as follows:
```
NAME                             READY   STATUS    RESTARTS   AGE
diag-collector-5c9d8968c-clnfr   1/1     Running   0          89s
```

## Collect diagnostic data
Diag can quickly capture the diagnostic data of TiDB cluster, including monitoring data, configuration information, etc.

### Usage scenario
The following scenarios apply to collecting diagnostic data using Diag:
- When there is a problem with the cluster, it is necessary to provide cluster diagnostic data to assist the technical support personnel in locating the problem when consulting PingCAP technical support.
- Keep cluster diagnostic data for later analysis.

> Attention
> For clusters deployed using TiDB Operator, collecting diagnostic data such as logs, configuration files, and system hardware information is not supported for the time being.

### Determine the data to be collected
For a detailed list of data collected by Diag, please refer to the [Clinic data collection instructions-Operator environment](https://clinic-docs.vercel.app/docs/getting-started/clinic-data-instruction-for-operator). It is recommended to collect complete monitoring data, configuration information and other data to improve diagnostic efficiency.

### Collect data
All operations of the Clinic Diag tool will be completed through the API.
- To see the full API definition doc, visit the node `http://${host}:${port}/api/v1` in web browser.
- To view the node IP, use the following command: `kubectl get node | grep node`
- To view the port number of the diag-collector service , use the following command: `kubectl get service -n tidb-admin`
```
NAME                 TYPE           CLUSTER-IP           EXTERNAL-IP   PORT(S)              AGE
diag-collector   NodePort   10.111.143.227   <none>            4917:31917/TCP   18m
```
In the example above:
  - The port to access the service from outside the Kubernetes cluster is `31917`.
  - The service type is `NodePort`. You can access the service by Kubernetes the IP address `${host}` and port number `${port}` of any host in the cluster.

1. Initiation of data collection requests
Initiate a data collection task through an API request:
```
curl -s http://${host}:${port}/api/v1/collectors -X POST -d '{"clusterName": "${cluster-name}","namespace": "${cluster-namespace}","from": "2022-02-08 12:00 +0800","to": "2022-02-08 18:00 +0800"}'
```

API call parameter description:
- `CluststerName`: TiDB cluster name
- `Namespace`: The namespace name of the TiDB cluster (not the namespace of the TiDB Operator)
- `Collector`: Optional parameter, you can configure the data type to be collected, and support `[monitor, config, perf]`. If this parameter is not configured, monitor and config data will be collected by default.
- `From` and `To`: the start and end time of acquisition respectively. `+0800` represents the time zone, and the supported time formats are as follows:
```
"2006-01-02T15:04:05Z07:00"
"2006-01-02T15:04:05.999999999Z07:00"
"2006-01-02 15:04:05 -0700"
"2006-01-02 15:04 -0700"
"2006-01-02 15 -0700"
"2006-01-02 -0700"
"2006-01-02 15:04:05"
"2006-01-02 15:04"
"2006-01-02 15"
"2006-01-02"
```

An example of the command output is as follows:
```
{
    "clusterName": "${cluster-namespace}/${cluster-name}",
    "collectors"
            "config",
            "monitor"
    ],
    "date": "2021-12-10T10:10:54Z",
    "from": "2021-12-08 12:00 +0800",
    "id": "fMcXDZ4hNzs",
    "status": "accepted",
    "to": "2021-12-08 18:00 +0800"
}
```

API return information description:
- `Date`: When the acquisition task was initiated.
- `Id`: The ID number of this task. In subsequent operations, this ID is the only information located to this task.
- `Status`: The current status of this task, accepted represents the acquisition task into the queue.

> Attention
> Returning the result of the command only means that the data collection task has started, and does not mean that the collection has been completed. To know whether the collection is complete, you need to check the status of the collection task through the next action.

2. View the status of data collection tasks
Get the status of the acquisition task through API request:
```
curl -s http://${host}:${port}/api/v1/collectors/${id}
{
    "clusterName": "${cluster-namespace}/${cluster-name}",
    "collectors": [
        "config",
        "monitor"
    ],
    "date": "2021-12-10T10:10:54Z",
    "from": "2021-12-08 12:00 +0800",
    "id": "fMcXDZ4hNzs",
    "status": "finished",
    "to": "2021-12-08 18:00 +0800"
}
```

Where `id` is the ID number of the task, in the above example, `fMcXDZ4hNzs`.

The return format of this step command is the same as that of the previous step (initiating a data acquisition request).

When the `status` of the task changes to `finished`, the data collection is complete.

3. View collected data set information
After completing the acquisition task, you can obtain the acquisition time and data size information of the data set through the API request:
```
curl -s http://${host}:${port}/api/v1/data/${id}
{
    "clusterName": "${cluster-namespace}/${cluster-name}",
    "date": "2021-12-10T10:10:54Z",
    "id": "fMcXDZ4hNzs",
    "size": 1788980746
}
```

With this command, you can only view the package size of the dataset, not the specific data. To view the content of the data, see [Optional Action: View Data Locally](https://clinic-docs.vercel.app/clinic-user-guide-for-operator.md#%E5%8F%AF%E9%80%89%E6%93%8D%E4%BD%9C%E6%9C%AC%E5%9C%B0%E6%9F%A5%E7%9C%8B%E6%95%B0%E6%8D%AE).

### Upload the dataset
When providing diagnostic data to PingCAP technical support personnel, you need to upload the data to Clinic Server, and then send its data link to the technical support personnel. [Clinic Server](https://clinic.pingcap.com) is the Cloud as a Service of Clinic Diagnostic Services, which provides more secure diagnostic data storage and sharing.

1. Initiate upload task
Package and upload the collected data set through API request:
```
curl -s http://${host}:${port}/api/v1/data/${id}/upload -X POST
{
        "date": "2021-12-10T11:26:39Z",
        "id": "fMcXDZ4hNzs",
        "status": "accepted"
}
```

Returning the command result only means that the upload task has started, and does not mean that the upload has been completed. To know whether the upload task has completed, you need to check the task status through the next action.

2. Check the status of the upload task
Check the status of the upload task through the API request:
```
curl -s http://${host}:${port}/api/v1/data/${id}/upload
{
    "date": "2021-12-10T10:23:36Z",
    "id": "fMcXDZ4hNzs",
    "result": "https://clinic.pingcap.com:4433/diag/files?uuid=ac6083f81cddf15f-34e3b09da42f74ec-ec4177dce5f3fc70",
    "status": "finished"
}
```

If the `status` changes to `finished`, the packaging and upload are complete. At this time, the result indicates that the Clinic Server views the link to this dataset, which is the data access link that needs to be sent to PingCAP technical support personnel.

> Optional action: view data locally
The collected data will be stored in the `/diag/collector/diag-${id}` directory of the Pod. You can enter the Pod to view this data by the following methods:
1. Get name of the Pod
```
kubectl get pod --all-namespaces  | grep diagtidb-admin
diag-collector-69bf78478c-nvt47               1/1     Running            0          19h
```

In this example, the name of the Diag Pod is `diag-collector-69bf78478c-nvt47`, and its namespace is `tidb-admin`.

2. Enter the pod and view the data
```
kubectl exec -n ${namespace} ${diag-collector-pod-name}  -it -- shcd  /diag/collector/diag-${id}
```
Where `${namespace}` needs to be replaced with the namespace name of the TiDB Operator (typically `tidb-admin`).

