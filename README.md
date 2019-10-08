# tidb-foresight

## 目录结构

- web: 前端 ui 界面
- api: api server, 对前端提供 restful 接口
- pioneer: 集群连通性检测工具，接收 invertory.ini，检测 api 所在机器和 invertory.init 里的所有机器是否正常连通
- collector: 诊断元信息收集器，通过 ansible 和应用层接口到集群负载机器上收集用于诊断的信息，供后续分析
- analyzer: 分析器，从诊断元信息中分析出可用于前端展示的信息，并插入数据库

## 开发指南

- 安装好所有依赖
    - yarn (nodejs > 10)
    - influxdb
    - prometheus
    - 诊断工具依赖的外部程序
        - 在每个节点上执行命令 `sudo yum install sysstat net-tools perf`
- 手动部署好 tidb-foresight
    - clone 源码到 `/tmp/tidb-foresight` 目录中
    - 配置系统服务，在 `/etc/systemd/system/` 目录下新建 
        - prometheus-9529.service
        - foresight-9527.service
    - `sudo systemctl start prometheus-9529.service`
    - root 权限下运行 `./deploy.sh`，会将源码目录下的前后端程序打包并拷贝到 `/home/tidb/tidb-foresight/` 目录下
- 源码发生修改后，更新 `/tmp/tidb-foresight` 仓库中的代码，然后
    - 运行 `./deploy.sh`，即可重新部署，会保留原来的诊断文件和 db
    - 可以按照需要删除部署目录下的 `sqlite.db` 等文件（夹）
    - 需要重置诊断工具，请删除整个部署目录 `/home/tidb/tidb-foresight/`，运行 `deploy.sh`
- 调试方法：
    - 打开浏览器，访问 `<本机 ip>:9527` 与前端交互
    - 查看运行状态 `systemctl status foresight`
    - 查看日志 `journalctl -u foresight | tail -n 20`

## Makefile 

注意：目前只支持 CentOS

* `make`  `make all`： 下载依赖并编译所有内容
* `make stop` 关闭正在运行的诊断工具服务
* `make start` 运行诊断工具服务
* `make install --prefix=` 把诊断工具代码编译，并放到 `prefix` 指定的目录下



