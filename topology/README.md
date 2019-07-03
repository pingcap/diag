# 集群连通性检测
场景：用户从前端页面新建集群，提交inventory.ini文件后，我们需要对inventory.ini所描述的集群进行验证，若ssh通道不通，则需要在前端提示用户打通通道，否则后续诊断无法进行

需要检测的内容：

- 使用ansible sdk到目标机执行whoami是否有返回值
- 使用ansible sdk带become参数到目标机执行whoami返回值是否为root
- 读取各个组件的启动脚本(deploy/scripts/run_xxx.sh)并获取他们监听的端口(其中任意一个)，用于后续确认进程

## 用法
`python topology.py inventory.ini`

返回内容：
```json
{
    "status": "success|exception",    // 表示本次检测任务的正常与否
    "hosts": [{                       // 若status为success则列表不为空
        "ip": "1.1.1.1",
        "user": "见说明",
        "components": [{
            "name": "tidb",
            "port": 4000              // 端口号用于标示进程
        }, {
            "name": "pd",
            "port": 2379
        }]
    }, {
        "ip": "1.1.1.2",
        "user": "见说明",
        "components": [{
            "name": "tidb",
            "port": 4000
        }]
    }]
}
```
- 若第一项检测不通过（无法连通），则user为空
- 若第二项失败，则user为第一项返回内容
- 若第二项成功，则user为root