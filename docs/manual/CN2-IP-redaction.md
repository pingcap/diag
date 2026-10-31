# CN2 实例 IP 脱敏版 Diag 使用说明

这份文档面向需要采集 TiDB 诊断数据的运维人员。
你可以按本文完成采集、实例 IP 脱敏、打包和上传。
本文说明当前 CN2 实例 IP 脱敏版的手动、自动、保留尾段和原地处理选项。
首次交付的 `d59432377a1f` 版本只有手动模式。
自动模式需要更新后的程序，运行 `./diag redact --help` 应能看到 `--redact-all-ips`。
日志 IPv4 隐藏模式需要 `--hide-log-ipv4`。
保留尾段、原地处理和本地对照表分别需要 `--keep-ipv4-octets`、`--redact-in-place` 和 `--ip-map-output`。
请运行新版帮助核对选项；旧包可能没有这些开关。

**当前功能处理 IP 文本，不会隐藏其他 SQL 参数值或业务数据。**
手动模式替换清单中的 IP；自动模式替换识别到的全部 IP。
自动模式可额外选择：日志正文中的 IPv4 改为 `?`，其余位置继续使用不同的替代 IP。
慢日志中的 SQL 文本也会参与扫描。

## 1. 工具做了什么

Diag 先按原有流程采集数据，再生成一份脱敏副本。
默认将原始采集目录和脱敏目录分别保存。
加上 `--redact-in-place` 后，会在原采集目录逐文件替换，最终保留一份采集数据。
服务器上的原始日志保持不变。

处理顺序如下：

```text
使用真实地址连接集群
  → 将数据采集到本地原始目录
  → 扫描文件内容、文件名和目录名
  → 按手动清单或自动发现结果建立统一映射
  → 将识别到的 IP 替换为固定的替代 IP
  → 复读脱敏副本，检查已知 IP 字面量残留
  → 使用者将脱敏副本打包并上传 Clinic
```

脱敏发生在采集结束后、打包加密前。
它不会在数据库写日志时修改内容，也不会修改集群配置。

`CN2` 是平凯 Clinic 的区域名称，上传地址为 `https://clinic.pingkai.cn`。
CN2 负责选择 Clinic 地址和加密证书。
`--redact-ips` 选择手动清单，`--redact-all-ips` 选择自动识别。

## 2. 准备程序和运行环境

### 2.1 选择压缩包

| 使用机器 | 压缩包后缀 | `uname -s` / `uname -m` 常见输出 |
| --- | --- | --- |
| Linux x86_64 运维服务器 | `linux-amd64.tar.gz` | `Linux` / `x86_64` |
| Linux ARM64 运维服务器 | `linux-arm64.tar.gz` | `Linux` / `aarch64` |

本次测试包提供 Linux AMD64 和 ARM64 两种版本。

可以先查看本机类型：

```bash
uname -s
uname -m
```

在存放压缩包的目录执行。Linux 示例：

以下命令使用首次交付的文件名。收到的文件名如果不同，请使用实际名称。

```bash
tar -xzf diag-cn2-ip-redaction-d59432377a1f-linux-amd64.tar.gz
cd diag-cn2-ip-redaction-d59432377a1f-linux-amd64
./diag --version
```

ARM64 机器请选择文件名中含 `linux-arm64` 的压缩包。
构建版本以 `--version` 和 `BUILD.txt` 为准。
后续命令均在解压后的程序目录中执行。

如果拿到了配套的 `SHA256SUMS`，可以在解压前核对压缩包：

```bash
shasum -a 256 -c SHA256SUMS
```

清单中的每个压缩包应各显示一行 `OK`。
旧清单可能包含多个平台；缺少的文件会提示缺失，需要核对已下载文件对应的一行。
Linux 上如果没有 `shasum`，可以使用 `sha256sum -c SHA256SUMS`。

### 2.2 保留完整的程序目录

| 文件或目录 | 用途 |
| --- | --- |
| `diag` | 采集、脱敏、打包和上传的主程序，保留原默认值 |
| `diag-redact` | 独立入口，默认自动脱敏、保留 IPv4 后两段和原地处理 |
| `info.toml` | Clinic 区域、上传地址和加密证书 |
| `scraper`、`insight` | 随包提供的辅助程序 |
| `profiles/`、`metrics/` | 采集模板和指标过滤配置 |
| `BUILD.txt` | 源码提交、分支、目标平台和构建版本 |
| `LICENSE`、`README.md` | 许可证和随包说明 |

`info.toml` 必须与 `diag` 放在同一目录。
如果只复制 `diag`，区域选择和打包可能失败。
详细说明可作为独立文档分发；随包 `README.md` 是交付时的简版说明。

### 2.3 确认采集条件

`./diag redact` 只处理本地文件，可以离线运行，也不需要 Clinic Token。
它会跳过 TiUP 仓库初始化，并关闭该命令的遥测报告。

采集、配置、打包和上传保留原版 Diag 的 TiUP 初始化要求。
运行机器需要已有的 TiUP 元数据。
例如，只有空的 `TIUP_HOME` 目录时，可能报缺少 `bin/root.json`。

采集 TiUP 部署的集群时，还需要本机已有的集群记录和 SSH 权限。
Diag 可能通过原有 TiUP 流程下载采集工具。
随包提供辅助程序不代表所有采集场景都能离线运行。

采集 Kubernetes 集群时，需要在获授权的采集机器上执行。
使用已批准的 kubeconfig，并核对其中的目标集群和命名空间。

### 2.4 从其他目录运行，或通过 TiUP 调用

`diag` 按可执行文件所在的目录查找 `info.toml`，不要求当前工作目录与程序目录相同。
保留整个解压目录后，可以通过绝对路径调用：

```bash
/opt/diag-cn2/diag collect my-cluster --redact-all-ips
```

这里的 `/opt/diag-cn2` 是示例，请替换为实际解压目录。
`--output` 等相对路径仍以执行命令时的工作目录为起点。

TiUP 支持用 `--binpath` 指定这份定制程序：

```bash
tiup --binpath /opt/diag-cn2/diag diag --version
tiup --binpath /opt/diag-cn2/diag diag collect my-cluster --redact-all-ips
```

先核对版本，确认包含这次定制提交。`info.toml` 和模板仍留在同一安装目录。
直接执行 `tiup diag collect` 会选择 TiUP 管理的版本，不会自动使用这里解压的定制版。
要用这种不带路径的方式分发定制版，还需要将版本发布到获授权的 TiUP 镜像。
本次测试包不修改已有 TiUP 安装，也不发布镜像。

### 2.5 独立的 diag-redact 入口

新版测试包增加 `diag-redact` 程序，复用相同的采集和脱敏实现。
原来的 `diag` 默认行为保持不变。两个入口的默认值如下：

| 参数 | `diag` | `diag-redact` |
| --- | --- | --- |
| `--exit-on-error` | `false` | `true`，用于采集命令 |
| `--redact-all-ips` | `false` | `true` |
| `--keep-ipv4-octets` | `0`，完整替代 | `2`，保留后两段 |
| `--redact-in-place` | `false` | `true` |
| `--ip-map-output` | `N`，不导出 | `N`，不导出；使用 `Y` 才导出 |
| `--redact-workers` | `4` | `4` |

这些默认值用于 `collect`、`collectk`、`collectdm` 和已有目录的 `redact` 命令。
`--redact-workers` 控制同时处理的文件数，范围为 1 到 256。短参数为 `-w`，例如 `-w 4` 等同于 `--redact-workers 4`。与 TiDB、TiKV 共用机器时可设为 1。扫描、脱敏和校验均使用该值。单个压缩包内部仍顺序处理；并发会增加临时磁盘空间和内存占用。此参数不限制整个进程的 CPU 核数。
包中同时提供定制 TiUP 时，可以在解压目录执行：

```bash
./tiup diag-redact collect tidb-test \
  --ip-map-output Y \
  --from '2026-10-30T11:01:00+08:00' \
  --to '2026-10-30T11:02:00+08:00' \
  --include 'system,config,monitor,log'
```

也可以直接执行 `./diag-redact collect ...`。
原地处理会替换本地采集内容，默认不保留完整原始副本。
需要副本和完整替代 IP 时，明确覆盖默认值：

```bash
./tiup diag-redact collect tidb-test \
  --redact-in-place=false \
  --keep-ipv4-octets 0 \
  --redact-output ./diag-redacted
```

普通终端运行时，`diag-redact` 的 Clinic 配置保存在 `~/.tiup/storage/diag-redact/diag.toml`。
原版配置仍使用 `~/.tiup/storage/diag/diag.toml`。首次打包或上传前，为新入口设置区域：

```bash
./tiup diag-redact config clinic.region CN2
```

定制 TiUP 的 `diag-redact` 命令只运行同目录里的程序，不从镜像下载，也不自动更新。
`tiup`、`diag-redact`、`info.toml` 和模板应保留在同一安装目录。
缺少程序时会报错，不会回退到官方 `diag`。
其他 TiUP 组件继续使用原有调用逻辑。

先用 `./tiup` 测试，不替换机器原有的 TiUP。
若希望直接输入 `tiup diag-redact`，可以只在当前终端调整 PATH：

```bash
export PATH="/实际解压目录:$PATH"
tiup diag-redact --version
```

关闭该终端后，PATH 的这次调整失效。原有 TiUP 和 Diag 安装文件没有覆盖。

### 2.6 用脚本安装到 PATH

新版合并包提供 `install.sh` 和 `INSTALL.md`。进入解压后的程序目录，执行：

```bash
sh ./install.sh --replace-tiup
```

脚本优先使用当前 TiUP 所在的 `bin` 目录；没有时使用 `/usr/local/bin`。
系统 `/usr/bin` 或 `/bin` 中的 TiUP 保留，新命令默认安装到 `/usr/local/bin`。
需要替换已有命令时会先备份，卸载时可恢复，不需要手动复制程序和资源。
安装位置、权限、验证和卸载步骤见 [安装说明](INSTALL.md)。

## 3. 选择自动识别或手动清单

### 3.1 不知道 IP 时，使用自动模式

自动模式使用 `--redact-all-ips`，无需提供 IP 清单。
程序先用正则表达式找疑似地址，再用 IP 解析器确认格式。
发现过程会增加一次全目录读取。复读校验只解码内容，不再次压缩。
它扫描受支持的文件内容、文件名、目录名和压缩内容。
IPv4 和 IPv6 都会识别，包括内网、公网、回环和未指定地址。

采集时使用：

```bash
./diag collect my-cluster \
  --exit-on-error \
  --output ./diag-original \
  --redact-all-ips \
  --redact-output ./diag-redacted
```

已有采集目录使用：

```bash
./diag redact ./diag-original \
  --redact-all-ips \
  --redact-output ./diag-redacted
```

`collectk` 和 `collectdm` 也支持这个开关。
`--redact-all-ips` 与 `--redact-ips` 互斥，每次选择一种模式。

自动模式通过格式识别地址，不能判断文本的实际业务含义。
例如，四段版本号 `1.2.3.4` 也是合法 IPv4 格式，因此可能替换。
如果需要保留这类内容，可以改用手动清单模式。
自动模式也不会隐藏域名、主机名、编码后的 IP 或其他业务数据。

### 3.2 保留 IP 最后一段，前三段固定为 10.0.0

IP 的“段”指点号之间的数字，不是字符串末尾的字符数。
例如，`10.12.34.56` 的后两段是 `34.56`，后一段是 `56`。

| 参数 | 原始 IP | 替代 IP 示例 |
| --- | --- | --- |
| `--keep-ipv4-octets 2` | `10.12.34.56` | `198.18.34.56` |
| `--keep-ipv4-octets 1` | `10.12.34.56` | `10.0.0.56` |

当前交付使用 `--keep-ipv4-octets 1`，保留最后一个点后面的完整数字。
例如，`172.16.34.56` 和 `192.168.8.57` 分别变为 `10.0.0.56` 和 `10.0.0.57`。
日志、慢日志中的 SQL、路径、拓扑和监控标签使用相同规则，端口保持不变。
这是部分隐藏，仍会暴露最后一段，不提供加密保护。
输出使用合法 IPv4，便于拓扑和监控解析，不会在结构中生成 `*.*.34.56`。
保留最后一段时，若原地址已经是 `10.0.0.x`，程序会报错，避免原始 IP 原样保留。
此时可以取消保留尾段选项，改用完整替代 IP 和本地对照表。
保留后两段的旧选项仍可使用，替代前缀是 `198.18`；原值相同时改用 `198.19`。
IPv6 仍使用完整替代地址。

```bash
./diag collect my-cluster \
  --exit-on-error \
  --output ./diag-original \
  --redact-all-ips \
  --keep-ipv4-octets 1 \
  --redact-output ./diag-redacted
```

新版允许不同 IPv4 的保留部分重复。
例如，`10.12.34.56` 与 `172.16.8.56` 在保留一段时都会变成 `10.0.0.56`。
`10.12.34.56` 与 `172.16.34.56` 在保留两段时都会变成 `198.18.34.56`。
这会让日志、拓扑和监控中不同实例的地址相同，监控序列也可能合并。
私有对照表仍逐条保留原始 IP，但一个替代 IP 可能对应多个原始 IP，无法唯一反查。
需要区分这些地址时，取消保留尾段选项，使用完整替代 IP。

文件和归档路径仍然禁止冲突，避免覆盖内容。
例如，两个不同 IP 目录都变成 `198.18.34.56/` 时，程序仍会报错。
副本模式不发布结果；原地模式会在预检查中停止，原始文件保持不变。

`--keep-ipv4-octets` 只支持 `1` 或 `2`。
它需要手动清单或自动模式，并与 `--hide-log-ipv4` 互斥。
如果日志正文都变为 `?`，就无法靠保留的尾段识别正文中原来的地址。

### 3.3 用本地对照表追溯真实 IP

`--ip-map-output` 现在只接收 `Y` 或 `N`，不再填写路径。
使用 `Y` 时导出真实 IP 与替代 IP 的对应关系；`N` 或省略参数时不导出。
文件权限为 `0600`，自动放在原始采集目录的同级目录中。
名称为 `<采集目录名>-ip-map.json`，沿用采集目录的随机名称，不另外生成随机数。
例如，`diag-tidb-test-h3hmHpvwqc0/` 对应同级的 `diag-tidb-test-h3hmHpvwqc0-ip-map.json`。
没有指定 `--output` 时，每次采集会生成不同目录名，对照表也会随之使用不同名称。
明确指定采集目录时，对照表使用该目录名。当前工作目录不会影响这个对应关系。
文件保存在诊断目录之外，已有同名文件不会覆盖。

```bash
./diag redact ./diag-original \
  --redact-all-ips \
  --ip-map-output Y \
  --redact-output ./diag-redacted
```

JSON 中的 `entries` 记录对应关系，例如：

```json
{
  "original_ip": "10.12.34.56",
  "redacted_ip": "198.18.0.1"
}
```

后端报告 `198.18.0.1` 的问题时，客户在本地查表，即可找到真实实例。
这份小文件不包含日志正文，空间占用与 IP 数量有关，远小于整份采集数据。
**对照表包含真实 IP，不能复制进诊断目录，也不能上传给后端。**

对照表会在数据改写前保存，避免原地处理后丢失对应关系。
导出成功不代表数据脱敏成功，仍需确认命令最终成功退出。
日志正文已经变成 `?` 时，对照表只能用于结构和监控中的替代 IP，无法恢复正文里每个问号原来的地址。

### 3.4 继续上传 Clinic，日志正文中的 IPv4 改成问号

同时使用 `--redact-all-ips` 和 `--hide-log-ipv4`：

```bash
./diag collect my-cluster \
  --exit-on-error \
  --output ./diag-original \
  --redact-all-ips \
  --hide-log-ipv4 \
  --redact-output ./diag-redacted
```

已有采集目录也可以处理：

```bash
./diag redact ./diag-original \
  --redact-all-ips \
  --hide-log-ipv4 \
  --redact-output ./diag-redacted
```

这个附加选项只与自动模式一起使用。单独使用或与手动清单一起使用会报错。

| 位置 | 原始内容 | 处理结果示例 |
| --- | --- | --- |
| 日志正文 | `peer=10.1.1.1:4000` | `peer=?:4000` |
| 慢日志 SQL | `server_ip='10.1.1.1'` | `server_ip='?'` |
| 文件路径 | `10.1.1.1/tidb.log` | `198.18.0.1/tidb.log` |
| 另一个实例的路径 | `10.1.1.2/tidb.log` | `198.18.0.2/tidb.log` |
| 拓扑、配置和监控标签 | `10.1.1.1:4000` | `198.18.0.1:4000` |

两个实例的日志正文都可能出现 `?`，所以不能再靠正文 IP 区分实例。
文件路径、结构中的引用和监控标签仍区分实例，避免路径覆盖和标签合并。
如果需要关联日志里的对端 IP 或定位真实地址，请优先使用完整替代 IP 或保留尾段，不启用日志问号选项。
IPv6 正文继续使用不同的替代 IPv6；IPv4 映射的 IPv6 写法也会整体替换成 `?`。

程序按原始文件名和目录识别日志。压缩文件先解码，再判断内部文件。
常见的 `.log`、`.log.1`、`.log.json`、`.stderr`、`.err` 和 RocksDB `LOG` 都会处理。
`log/`、`logs/`、`slowlog/`、`slowlogs/` 和 TiUP 审计目录中的文本也会处理。
归档中的文件会保留父目录语境，例如 `logs/bundle.zip/events.txt` 会按日志处理。

拓扑文件、已知配置文件和监控目录优先保留结构映射。
例如，`cluster.json`、`config.json`、YAML、TOML 和 `monitor/` 中的数据不会改成问号。
无法按这些规则识别的自定义日志文件，仍会使用不同的替代 IP。
因此，启用选项后也应抽查实际日志的处理结果。

处理结束会输出 `Log body IPv4 replaced with ?:`，后面是正文替换次数。
程序还会复读日志，发现残留的可识别 IPv4 时拒绝生成最终副本。

这项设计保留诊断包的实例结构，已经验证加密打包、解包及文件引用。
实际 Clinic 导入和页面展示仍需用测试数据验收。

### 3.5 已知实例 IP 时，使用手动清单

手动模式使用 `--redact-ips`，需要包含所有待隐藏的实例 IP。
未列出的地址保持不变。

可以从集群拓扑或现有运维平台取得清单。
对于 TiUP 部署的集群，可以在已有权限的机器上查看：

```bash
tiup cluster display my-cluster
```

根据需要，纳入 TiDB、TiKV、PD、TiFlash 和监控节点的 IP。
同一实例如果同时有内网和外网地址，需要分别列出。
如果还需要隐藏客户端地址，也需要把这些地址加入清单。

支持逗号分隔，也支持重复指定参数：

```bash
./diag redact ./diag-original \
  --redact-ips '10.1.1.1,10.1.1.2' \
  --redact-ips '2001:db8:1234::1' \
  --redact-output ./diag-redacted
```

这个命令是已有采集目录的用法。先采集的操作见下一节。

| 清单内容 | 是否支持 |
| --- | --- |
| IPv4，例如 `10.1.1.1` | 支持 |
| IPv6，例如 `2001:db8:1234::1` | 支持 |
| 带端口，例如 `10.1.1.1:4000` | 不支持，只填写地址 |
| 网段，例如 `10.1.1.0/24` | 不支持，需要列出具体地址 |
| 域名或主机名 | 不支持 |
| 带接口区域的地址，例如 `fe80::1%eth0` | 清单中不支持，填写 `fe80::1` |

日志里的端口会保留。
例如，清单填写 `10.1.1.1`，就能处理日志中的 `10.1.1.1:4000`。
支持格式中的 IPv6 大小写、压缩写法和展开写法会按地址识别。

## 4. 采集时生成脱敏副本

### 4.1 TiUP 部署的 TiDB 集群

将示例集群名、时间和 IP 清单替换为实际值：

```bash
./diag collect my-cluster \
  --from '2026-10-21T09:00:00+08:00' \
  --to '2026-10-21T10:00:00+08:00' \
  --include 'system,config,monitor,log.std,log.slow' \
  --exit-on-error \
  --output ./diag-original \
  --redact-ips '10.1.1.1,10.1.1.2' \
  --redact-output ./diag-redacted
```

时间中的 `+08:00` 表示 UTC+8 时区。这里的日期和时间仅作示例。
`log.std` 采集常规日志，`log.slow` 采集慢日志。
脱敏选项只处理已经采集到的内容，不会额外采集日志或 SQL。

| 参数 | 含义 |
| --- | --- |
| `--from`、`--to` | 采集时间范围 |
| `--include` | 本次需要的采集项；上例包含慢日志 |
| `--exit-on-error` | 遇到采集错误就停止，先处理采集问题 |
| `--output` | 保存原始数据的目录 |
| `--redact-ips` | 手动模式，需要隐藏的实例 IP 清单 |
| `--redact-all-ips` | 自动发现并替换全部识别到的 IP，与手动清单互斥 |
| `--hide-log-ipv4` | 附加选项，日志正文 IPv4 改为 `?`，需要自动模式 |
| `--keep-ipv4-octets` | `1` 保留最后一段，前缀为 `10.0.0`；`2` 保留后两段。允许重复替代 IP，但路径不能冲突 |
| `--redact-in-place` | 在原采集目录替换内容，不保存完整原始副本 |
| `--ip-map-output Y/N` | `Y` 在采集目录旁生成同名对照表；`N` 不导出，默认 `N` |
| `--redact-output` | 保存脱敏副本的新目录 |

`--include` 会指定本次采集范围。修改清单时，要保留需要的采集项。
使用自定义 `--profile` 时，还应核对模板和最终采集范围。

原始目录与脱敏目录必须不同。
脱敏输出目录必须在原始目录之外，父目录必须已经存在。
输出目录本身必须不存在。

如果省略 `--redact-output`，默认生成 `<原始目录名>-redacted`。
例如，`--output ./diag-original` 对应 `./diag-original-redacted`。
如果原始目录名包含指定 IP，默认脱敏目录名也会替换这个 IP。

### 4.2 Kubernetes 和 DM 集群

`collectk` 和 `collectdm` 支持相同的脱敏参数。
具体采集选项以各命令的 `--help` 为准。

Kubernetes 示例，应在获授权的采集机器上执行：

```bash
./diag collectk my-tidbcluster \
  --namespace my-namespace \
  --kubeconfig /path/to/approved-kubeconfig \
  --include 'monitor.metric,log.std,log.slow' \
  --exit-on-error \
  --output ./diag-k8s-original \
  --redact-ips '10.1.1.1,10.1.1.2' \
  --redact-output ./diag-k8s-redacted
```

`collectk` 使用的集群名称来自 TiDB Operator 的 TidbCluster 资源。
kubeconfig 的当前上下文应指向目标集群。
它的默认采集范围与 `collect` 不同，不能据此假定两种模式的文件完全相同。

DM 示例：

```bash
./diag collectdm my-dm-cluster \
  --exit-on-error \
  --output ./diag-dm-original \
  --redact-ips '10.1.1.1,10.1.1.2' \
  --redact-output ./diag-dm-redacted
```

DM 默认采集项不包含 `log.slow`。日志类型和采集范围需要按 DM 场景核对。

### 4.3 确认脱敏成功

新版在终端实时刷新脱敏进度，例如：

```text
IP redaction: Scanning IPs ... / [=====     ]  50% | 20/40 files | 12s
```

自动模式依次显示 `Scanning IPs`（扫描 IP）、`Redacting`（脱敏）和 `Verifying`（校验）。
手动模式跳过扫描 IP；原地模式还会显示预检查和原地改写阶段。
每个阶段单独计数，百分比按该阶段完成的外层文件数量计算。
单个大文件处理期间，文件数可能暂时不变；进度不能用于推算剩余时间。
最后一个文件完成后可能仍需其他处理，因此阶段完成前最多显示 `99%`。
进度不显示文件名或原始 IP。
输出重定向到文件或管道时，每个阶段只输出开始和结束提示。

`The collected data has been stored in ...` 只表示采集完成。
脱敏仍在运行时不需要按 Ctrl+C；请等待下面的完成提示和命令成功退出。

自动模式还会输出发现的不同 IP 数量，例如 `Automatic IP redaction: 9 unique IPs mapped.`。
发现数量为 `0` 时，不能据此认定所有敏感信息已经隐藏。

两种模式成功时，都会输出类似内容：

```text
IP redaction complete: 4 files, 8 replacements.
Package ONLY this directory: /path/to/diag-redacted
The original collection still contains real IPs.
```

`files` 是处理的外层文件数。`replacements` 包括内容和路径中的替换次数。
同一个 IP 多次出现会多次计数，所以这个数字不是实例数。

只有命令成功退出，并输出上述完成提示时，才使用脱敏目录。
**后续打包只填写脱敏目录，原始目录继续留在本地。**

替换次数为 `0` 时，需要核对 IP 清单和采集范围。
它可能表示清单与内容不匹配，也可能表示本次数据确实没有这些地址。

### 4.4 数据很大时，只保留一份采集目录

默认方式的磁盘占用通常接近“原始目录＋脱敏目录”。
生成 `.diag` 后，还会额外占用诊断包的空间。
需要减少完整副本时，显式使用 `--redact-in-place`。

下面的命令保留 IPv4 最后一段，前三段固定为 `10.0.0`，并直接改写 `diag-data`：

```bash
./diag collect my-cluster \
  --exit-on-error \
  --output ./diag-data \
  --redact-all-ips \
  --keep-ipv4-octets 1 \
  --redact-in-place
```

也可以对已经采集完成的目录执行：

```bash
./diag redact ./diag-data \
  --redact-all-ips \
  --keep-ipv4-octets 1 \
  --redact-in-place
```

不加 `--keep-ipv4-octets` 时，会使用完整替代 IP。
如需对照表，可以另外加 `--ip-map-output Y`，程序自动生成与采集目录名匹配的同级文件。
`--redact-in-place` 与 `--redact-output` 互斥。
顶层目录名保持不变，目录内的 IP 文件名和子目录名会同步改写。
请给顶层目录使用不含真实 IP 的名称。

**原地处理会替换本地原始采集内容，不保留完整备份，不能承诺整批回滚。**
使用前应停止对采集目录的其他写入。服务器上的日志不受影响。
程序先检查文件格式和目标路径，再逐文件写临时内容、替换原文件，并在最后复读检查。

空间占用大致为：

```text
处理期间：一份采集数据＋当前文件的临时副本＋归档内部临时文件
处理完成：一份脱敏后的采集目录
继续打包：一份脱敏目录＋新生成的 .diag 文件
```

临时文件不是零空间。单个文件占目录大部分时，处理该文件的峰值仍可能接近两份大小。
归档内部处理也可能需要较大的临时空间，系统临时目录和采集磁盘都应预留容量。
这个选项解决“保留两份完整采集目录”的问题，不保证峰值固定低于某个比例。
如果最终只想留下 `.diag`，需要在确认包完整后自行安排清理采集目录，程序不会自动删除它。

预检查失败时，原始内容保持不变。
开始逐文件替换后失败或中断，目录可能同时含有原始和脱敏内容。
此时保留 `.redact-in-progress` 标记，新版 Diag 会拒绝继续脱敏或打包该目录，包括 `-y` 自动确认。
不要删除标记强行上传，也不要用其他打包工具绕过检查。
应在新的目录重新采集；如启用对照表导出，新采集目录会对应新的对照表文件名。

成功时标记会移除，命令会打印实际完成的目录。
原地模式后续直接打包这个目录：

```bash
./diag config clinic.region CN2
./diag package ./diag-data -o ./diag-data.diag
```

## 5. 处理已经采集的目录

如果已经用官方 Diag 采集完数据，可以生成脱敏副本：

```bash
./diag redact ./diag-original \
  --redact-ips '10.1.1.1,10.1.1.2' \
  --redact-output ./diag-redacted
```

处理前应确认采集已经结束，且没有其他进程继续写入目录。
如果仍有 `.collect.lock`，程序会拒绝处理。
先核对采集状态和数据完整性，再处理锁文件问题。

输入应是采集目录，例如包含 `cluster.json`、日志和监控数据的目录。
这个命令不接收加密 `.diag` 包作为输入。
如果只有加密包，需要回到原始采集目录处理，或者另行准备获授权的解密流程。

`redact` 本身也能处理只有日志的普通目录。
但将结果打成标准诊断包时，仍需要原版 Diag 要求的 `cluster.json` 等材料。

## 6. 打包并上传到 CN2

### 6.1 先选择 CN2，再打包

```bash
./diag config clinic.region CN2
./diag package ./diag-redacted -o ./diag-redacted.diag
```

第一个命令将当前 Diag 区域配置设为 CN2。
第二个命令将脱敏目录打成加密诊断包。
这两步都不会上传数据。

打包时填写明确的输入目录，避免让程序自动选择其他采集目录。
如果 `diag-redacted.diag` 已经存在，请使用新的输出文件名。

### 6.2 提供 Token 并上传

上传需要平凯 Clinic 的有效 Token。
可以沿用已配置的 `clinic.token`，也可以设置 `CLINIC_TOKEN` 环境变量。
环境变量存在时，上传命令优先使用环境变量中的值。

下面是在 Bash 或 zsh 中临时输入 Token 的方式。
输入时不回显，也不把实际值写进命令文本：

```bash
printf '请输入 Clinic Token：'
read -r -s CLINIC_TOKEN
printf '\n'
export CLINIC_TOKEN
./diag upload ./diag-redacted.diag
unset CLINIC_TOKEN
```

原始目录、脱敏目录和诊断包的关系如下：

| 路径 | 内容 | 用途 |
| --- | --- | --- |
| `diag-original/` | 含真实 IP 的原始采集数据 | 留在本地 |
| `diag-redacted/` | 按所选模式替换 IP 后的副本 | 作为打包输入 |
| `diag-redacted.diag` | 从脱敏目录生成的加密包 | 作为上传输入 |

执行 `upload` 才会上传数据。
上传成功后，仍需在 Clinic 页面检查重建状态和日志、监控展示。
当前版本已验证标准包的加密和解包，实际客户数据的页面验收仍需完成。

## 7. 替换规则和覆盖范围

### 7.1 同一个 IP 使用同一个替代值

程序先整理手动清单，或完整扫描目录取得自动发现的地址，再建立统一映射。
同一个 IP 在日志、配置、路径和监控标签中使用同一个替代值。
端口保持不变，便于关联同一实例的诊断数据。

| 原始内容 | 脱敏内容示例 |
| --- | --- |
| `10.1.1.1:4000` | `198.18.0.1:4000` |
| `10.1.1.2:20160` | `198.18.0.2:20160` |
| `logs/10.1.1.1/tidb.log` | `logs/198.18.0.1/tidb.log` |
| `[2001:db8:1234::1]:4000` | `[2001:db8::1]:4000` |

默认完整替代模式的 IPv4 使用 `198.18.0.0/15` 中的值。保留最后一段时使用 `10.0.0.x`。
IPv6 使用 `2001:db8::/32` 中的替代值。
示例中的结果对应示例清单，实际值取决于完整清单。
同一地址集合下，参数顺序和文件遍历顺序不会改变映射。
清单或自动发现的地址集合变化时，映射可能变化。
自动模式会避开输入中已有的地址，防止替代 IP 与原始 IP 混淆。

默认不输出或保存原始 IP 对照表。指定 `--ip-map-output Y` 后，才会在采集目录旁写入同名私有对照表。
默认副本模式保留原始数据；原地模式会替换这些内容。
替代 IP 只用于诊断数据中的实例标识，不应用于连接真实集群。

### 7.2 扫描整个目录中的受支持内容

| 位置或格式 | 处理方式 |
| --- | --- |
| 普通日志、慢日志 | 扫描文本，按所选模式替换 IP |
| SQL Statement 文本 | 按所选模式替换 IP，其他 SQL 内容保留 |
| JSON、YAML、TOML 和拓扑 | 扫描文本，按所选模式替换 IP |
| 监控标签 | 在受支持的监控文件内容中按所选模式替换 IP |
| 文件名和目录名 | 同步替换 IP |
| gzip、zstd | 解压处理后重新压缩 |
| tar、zip | 处理归档条目名称和受支持的内部内容 |

压缩格式按文件内容识别。
因此，扩展名为 `.json` 的 zstd 监控文件也能处理。
嵌套压缩和归档也会递归处理，但存在层数和资源上限。

gzip 的原始文件名、注释和扩展字段会清除。
tar 和 zip 的相关扩展元数据会清理或重新生成。
这些处理有助于避免旧归档元数据携带 IP。

仅扫描目录中已有的数据，不会因为启用脱敏而额外采集 SQL 或慢日志。
日志里只有主机名时，IP 清单也不会把这个主机名隐藏。

### 7.3 慢日志和 SQL 参数值的边界

当前工具按文本中的 IP 处理内容，不解析 SQL 语法。
慢日志、错误日志和 SQL Statement 中的 IP，都会按所选模式处理。

例如，原始 SQL 为：

```sql
SELECT * FROM users
WHERE server_ip = '10.1.1.1' AND phone = '13800000000';
```

使用默认替代 IP 模式，且目标地址参与替换时，处理后为：

```sql
SELECT * FROM users
WHERE server_ip = '198.18.0.1' AND phone = '13800000000';
```

如果 SQL 位于识别到的日志文件中，并启用日志 IPv4 隐藏模式，结果为：

```sql
SELECT * FROM users
WHERE server_ip = '?' AND phone = '13800000000';
```

手机号仍然保留。
字符串、数字、姓名、订单号、表名和列名也不会因为是 SQL 内容就自动隐藏。
如果某个业务字段的值符合所选模式的 IP 条件，工具也会替换这个值。

当前没有“SQL 参数脱敏”开关，也没有将 SQL 改写为参数化语句的功能。
如果交付要求包含业务数据脱敏，需要另行处理并验收，不能只根据 IP 脱敏成功作判断。

## 8. 检查结果和处理失败

### 8.1 程序会复读脱敏副本

复制和替换完成后，程序会重新读取输出内容。
复读包括受支持压缩格式的内部文件和条目名称。
手动模式检查清单中的 IP 字面量残留。
自动模式还检查输出地址是否全部来自替代值集合。
日志 IPv4 隐藏模式另外检查识别到的日志正文中是否残留 IPv4。
如果扫描后又出现新 IP，程序会拒绝生成最终副本。

复读通过后，程序才公布最终脱敏目录。
副本模式失败时，会清理本次临时副本，原始目录仍保留。
原地模式开始改写后可能留下混合数据，失败标记会阻止打包，需在新目录重新采集。

这项检查覆盖受支持的 IP 文本写法。
Base64、十六进制、整数 IP 和 JSON Unicode 转义等编码形式不在支持范围内。
手动模式不会处理未列出的 IP。两种模式都不保证隐藏域名、编码地址和其他业务数据。

如果机器上有 `rg`，可以辅助检查普通文本：

```bash
rg --hidden --no-ignore -n -F \
  -e '10.1.1.1' -e '10.1.1.2' ./diag-redacted
```

这条命令不解压归档，也不等同于程序的复读检查。
没有匹配结果只表示这次文本检索没有命中。
检索命中也需要核对，例如 `10.1.1.10` 可能包含检索串 `10.1.1.1`。

### 8.2 常见错误

| 报错内容或现象 | 原因和处理方式 |
| --- | --- |
| `redact-ips requires literal IPv4 or IPv6 addresses` | 清单含端口、网段或域名。改为具体 IP |
| `redact-ips cannot be empty` | 参数已填写，但清单为空。补充实际 IP |
| `redact-output requires redact-ips or redact-all-ips` | 只设置了输出目录。还需要选择一种模式 |
| `redact-all-ips and redact-ips are mutually exclusive` | 同时使用了两种模式。保留其中一种 |
| `hide-log-ipv4 requires redact-all-ips` | 日志 IPv4 隐藏选项需要同时启用自动模式 |
| `IPv4 literal remains in a question-mark log body` | 日志复读发现残留。保留错误和文件类型以便排查 |
| `unrecognized IP appeared after discovery` | 预扫描后发现新地址。确认目录停止写入，再重新运行 |
| `redact-in-place and redact-output are mutually exclusive` | 原地模式直接处理输入，移除脱敏输出目录参数 |
| `in-place redaction is incomplete` | 保留失败标记，在新目录重新采集，禁止打包当前目录 |
| 旧版提示 `IPv4 suffix collision` | 旧版拒绝重复尾段；新版允许重复替代 IP，但仍拒绝文件和归档路径冲突 |
| `IPv4 already uses the fixed 10.0.0 prefix` | 原地址已经是 `10.0.0.x`，改用完整映射和本地对照表 |
| `ip-map-output must be outside` | 自动对照表位置与诊断目录重叠，检查采集目录和脱敏输出目录的命名 |
| `ip-map-output must be Y or N` | 参数仍用了旧版路径写法，改为 `Y` 或 `N` |
| `ip-map-output exists` | 本采集目录已有对应的对照表，程序拒绝覆盖；检查是否重复处理同一目录 |
| `redaction output already exists` | 输出已存在。使用新目录名，保留已有结果 |
| `redaction output must be outside the input directory` | 输出位于原始目录内部。改为并列目录 |
| `collection is incomplete or still running` | 存在 `.collect.lock`。核对采集状态和完整性 |
| `unsupported binary` 或 `non-UTF-8` | 文件包含当前不支持的内容。按文件定位格式 |
| `unsupported or malformed compressed/encrypted file` | 压缩格式不支持、文件损坏，或输入了加密材料 |
| `symlinks and special files` | 输入包含符号链接或特殊文件。需要核对材料来源和用途 |
| `replacement IP already exists` 或 `path collision` | 替代地址或路径发生冲突。保留原始数据，记录错误以便调整方案 |
| `source IP literal remains` 或 `ambiguous text context` | 字面量仍有残留或文本上下文不明确。定位内容后处理 |
| `root.json` 缺失 | 原版命令初始化 TiUP 失败。回到已有 TiUP 环境执行 |
| CN2 区域不可用 | 核对 `info.toml` 位置，以及是否运行了专用版本 |

遇到不支持的内容时，不要把原始目录当作脱敏结果上传。
可以先确认排障所需材料，再调整采集范围重新采集。
保留错误信息和文件类型，供后续增加对应格式支持。

原始 Prometheus 数据块和某些 protobuf 性能分析文件属于二进制内容，可能无法处理。
需要这些材料时，先用测试数据确认支持范围。

### 8.3 资源上限

| 项目 | 上限或行为 |
| --- | --- |
| 压缩或归档嵌套 | 最多 8 层 |
| 目录项与归档条目计数 | 最多 1,000,000 个 |
| 不同 IP 数量 | 最多 65,536 个 |
| 文本展开总量 | 每次处理最多 64 GiB |
| 连续文本标记 | 受内部流式缓冲区限制，过长会报错 |
| 原始文件和目录 | 副本模式保留；原地模式替换内容 |
| 输出已有内容 | 拒绝覆盖 |

副本模式会同时占用原始目录、完整副本和归档临时文件的空间。
原地模式省去完整副本，但仍需要当前文件和归档的临时空间。
处理大型目录前，应按所选模式和最大文件预留容量。

## 9. 先用小样例离线试运行

这个例子只创建两个普通文本文件，不连接集群，也不上传数据。
在程序目录中执行，且确保 `redact-demo` 尚不存在：

```bash
mkdir -p ./redact-demo/input
cat > ./redact-demo/input/tidb.log <<'EOF'
peer=10.1.1.1:4000
EOF
cat > ./redact-demo/input/tidb_slow_query.log <<'EOF'
# User@Host: root[root] @ 10.1.1.1 []
SELECT * FROM users WHERE server_ip = '10.1.1.1' AND phone = '13800000000';
EOF
./diag redact ./redact-demo/input \
  --redact-all-ips \
  --hide-log-ipv4 \
  --redact-output ./redact-demo/output
cat ./redact-demo/output/tidb.log
cat ./redact-demo/output/tidb_slow_query.log
```

输出文件中的主要内容应为：

```text
peer=?:4000
# User@Host: root[root] @ ? []
SELECT * FROM users WHERE server_ip = '?' AND phone = '13800000000';
```

这个样例自动识别地址，验证普通日志和慢日志正文中的 IPv4 问号替换。
手机号保留，用来展示当前功能的边界。
样例没有 `cluster.json`，只能用于测试脱敏，不能直接打成标准诊断包。
重复试运行时，需要选择新的脱敏输出目录。

## 10. 版本来源和验证记录

首次交付二进制对应源码提交 `d59432377a1f9f832c64e04da440ae4fe68e539c`。
新版以随包 `BUILD.txt` 和 `./diag --version` 中的提交为准。
源码分支为 `codex/cn2-instance-ip-redaction-20261021`，基于现有 CN2 分支。
首次版本支持手动 IP 清单，后续在同一 PR 中补充自动识别。
两种模式均不包含 SQL 参数脱敏。
后续增加日志 IPv4 问号附加选项，诊断结构继续保留不同替代 IP。

PR: https://github.com/tao1020/diag/pull/1

2026-10-21 的验证结果如下：

| 检查 | 结果 |
| --- | --- |
| Go 1.25.5 全量测试、`go vet` 和静态检查 | 通过 |
| 脱敏数据的加密打包和解包测试 | 通过，使用生成的测试证书 |
| Mac ARM64 程序离线脱敏和 CN2 打包 | 通过 |
| Linux amd64 程序断网脱敏 | 在 Colima 和 GitHub Linux CI 中通过 |
| 本地历史采集目录 | 1,417 个文件，1,850 处替换，通过 |
| 自动模式处理同一本地目录 | 识别 14 个不同 IP，1,867 处替换，通过 |
| 两个平台的交付压缩包校验 | SHA-256 校验通过 |
| 真实客户集群采集与 Clinic 页面验收 | 尚未完成 |

这些检查验证了现有实现和测试样本，不能替代对实际客户数据的验收。

自动模式的单次本地样本耗时约 56 秒。
样本有 1,417 个文件，文件内容合计约 5.34 MiB，包含压缩文件。
计时包含发现、复制、改写和复读，不包含采集、最终打包和上传。
这次测试不能用于推算 GB 级诊断包的耗时。

## 11. 开发者重新构建

使用 Go 1.25 系列和仓库依赖，在仓库根目录执行：

```bash
bash scripts/build-cn2-redacted.sh
```

构建前应提交并验证代码。脚本会拒绝打包与当前提交不一致的已跟踪文件。
输出目录为 `package/cn2-ip-redaction`。
当前脚本生成 Linux AMD64、ARM64 测试压缩包，以及 `SHA256SUMS`。

将带本地入口的定制 TiUP 源码准备好并提交后，可以生成包含两个项目的测试包：

```bash
bash scripts/build-local-tiup-redact.sh /实际定制TiUP源码目录
```

输出目录为 `package/diag-redact-local`，文件名同时包含 Diag 和 TiUP 的源码短标识。
随包 `BUILD.txt` 和 `TIUP-BUILD.txt` 分别记录两个项目的版本，脚本拒绝打包未提交的已跟踪改动。

重新构建后的文件名包含当次源码提交的短标识，可能与本文已交付文件名不同。
压缩包中的 `BUILD.txt` 记录当次提交和构建平台。
程序依据文件内容识别压缩格式；实现见 [脱敏模块](../../pkg/redact/redact.go)。
命令参数及采集后的调用见 [脱敏命令](../../cmd/diag/command/redact.go)。
