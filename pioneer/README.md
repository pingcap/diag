# 集群连通性检测
场景：用户从前端页面新建集群，提交inventory.ini文件后，我们需要对inventory.ini所描述的集群进行验证，若ssh通道不通，则需要在前端提示用户打通通道，否则后续诊断无法进行

需要检测的内容：

- 解析 inventory.ini 来判断当前的 ansible_user 并使用此 user 来做相关 ansible 操作
- 通过 ansible sdk 调用 ping 模块判断 ssh 是否可以连通
- 使用 ansible sdk 带 become 参数到目标机执行 whoami 判断 user 是否有 sudo 权限
- 读取各个组件的启动脚本(deploy/scripts/run_xxx.sh)并获取他们监听的端口(其中任意一个)，用于后续确认进程

## 用法
`python pioneer.py inventory.ini`

返回内容：

```json
{
    "cluster_name": "test-cluster",
    "status": "exception",                  // "success|exception"，表示本次检测任务的正常与否，host 或者 components 中存在 exception 状态时，该值为 exception
    "message": "Fail list: [u'10.0.1.2', u'10.0.1.3']",
    "hosts": [
        {
            "status": "success",            // "success|exception"，表示和对应机器是否能够 ssh 连通
            "ip": "10.0.1.1",
            "enable_sudo": true,            // "true|false"，表示 user 用户是否有 sudo 权限
            "user": "tidb",
            "components": [
                {
                    "status": "success",    // "success|exception"，当从相关文件获取端口失败，状态为 exception，此时 port 变为 message，用来记录错误信息
                    "deploy_dir": "/data/deploy",
                    "name": "node_exporter",
                    "port": "9100"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "blackbox_exporter",
                    "port": "9115"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "prometheus",
                    "port": "9090"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "pushgateway",
                    "port": "9091"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "pd",
                    "port": "2379"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "status_port": "10080",
                    "name": "tidb",
                    "port": "4000"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "grafana",
                    "port": "3000"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "tikv",
                    "port": "20160"
                }
            ],
            "message": ""
        },
        {
            "status": "success",
            "ip": "10.0.1.2",
            "enable_sudo": true,
            "user": "tidb",
            "components": [
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "node_exporter",
                    "port": "9100"
                },
                {
                    "status": "exception",
                    "message": "cat: /data/deploy/scripts/run_blackbox_exporter.sh: No such file or directory",
                    "deploy_dir": "/data/deploy",
                    "name": "blackbox_exporter"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "tikv",
                    "port": "20160"
                }
            ],
            "message": ""
        },
        {
            "status": "success",
            "ip": "10.0.1.4",
            "enable_sudo": true,
            "user": "tidb",
            "components": [
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "node_exporter",
                    "port": "9100"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "blackbox_exporter",
                    "port": "9115"
                },
                {
                    "status": "success",
                    "deploy_dir": "/data/deploy",
                    "name": "tikv",
                    "port": "20160"
                }
            ],
            "message": ""
        },
        {
            "status": "exception",
            "ip": "10.0.1.3",
            "message": "Failed to connect to the host via ssh",
            "user": "tidb",
            "components": []
        }
    ]
}
```
