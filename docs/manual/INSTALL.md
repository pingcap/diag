# 安装本地 TiUP 和 Diag Redact

安装包提供 Linux x86_64 和 ARM64 版本。请根据 `uname -m` 选择对应版本。
安装包包含定制 TiUP、独立的 `diag-redact` 和配套资源。
安装脚本不会修改 TiUP 镜像、已有 Diag 组件、Clinic 配置、采集目录或 IP 对照表。

## 1. 解压并安装

解压 ZIP，再解压其中的 `tar.gz`，进入包含 `tiup`、`diag-redact` 和 `install.sh` 的目录。
执行下面一条命令即可，不需要手动复制程序：

```bash
sh ./install.sh --replace-tiup
```

位置选择规则：

- 当前 PATH 已有 TiUP 时，优先使用它所在的 `bin` 目录，通常不需要 export。
- 没有 TiUP 时，默认使用 `/usr/local/bin`。
- 若原 TiUP 位于 `/usr/bin` 或 `/bin`，默认改用 `/usr/local/bin`，保留系统包管理器的文件。

首次替换已有命令需要 `--replace-tiup`。
脚本会先将它改名为同目录的 `tiup.before-diag-redact`，再创建新命令链接。
不加这个参数时，遇到已有 TiUP 会报错，不会覆盖。
已有备份也不会覆盖；重复安装和升级会保留首次备份。

如果目标目录没有写权限，需要由管理员运行。
也可以明确指定常见的系统安装位置：

```bash
sudo sh ./install.sh --prefix /usr/local --replace-tiup
```

`sudo` 可能改变 PATH，因此使用它时建议明确指定 `--prefix`。
不要把新程序手动复制到 Yum/RPM 管理的 `/usr/bin`。
脚本不会自动提权，也不会修改 shell 启动文件。

## 2. 验证并使用

```bash
command -v tiup
tiup diag-redact --version
tiup diag-redact collect --help
```

核对安装脚本最后打印的命令位置。如果当前终端缓存了旧位置，先运行 `hash -r`，再验证。
自定义的 `--prefix` 如果不在 PATH 中，脚本会提示；可以使用打印的绝对路径。
默认选择常见 PATH 目录，通常不需要手动 export。

采集示例：

```bash
tiup diag-redact collect tidb-test \
  --ip-map-output Y \
  --from '2026-10-30T11:01:00+08:00' \
  --to '2026-10-30T11:02:00+08:00' \
  --include 'system,config,monitor,log'
```

默认启用遇错退出、自动 IP 脱敏、保留 IPv4 后两段和原地处理。
默认同时处理 4 个文件。使用 `-w 1` 或 `--redact-workers 1` 可降低并发数。
原地处理不保留完整原始副本。`--ip-map-output Y` 导出对照表，`N` 或省略参数则不导出。
文件自动放在采集目录旁边，名称为 `<采集目录名>-ip-map.json`，不能上传。
新入口的 Clinic 配置单独保存；首次打包或上传前设置区域：

```bash
tiup diag-redact config clinic.region CN2
```

## 3. 文件安装在哪里

以 `--prefix /usr/local` 为例：

```text
/usr/local/bin/tiup
  → /usr/local/lib/diag-redact/<Diag版本>-<TiUP版本>/tiup
```

版本目录中保留全部发布资源，包括 `diag-redact`、`info.toml`、采集模板和说明文档。
脚本不会将解压目录里额外生成的日志、采集数据或私有对照表复制进去。
安装前后都会核对发布文件的 SHA-256。已安装的同一版本如果被修改，脚本拒绝覆盖。

升级时，从新包的目录重新运行安装命令。
旧版本目录保留，新命令链接会切换到新版本。
同一位置一次只允许一个安装脚本运行；发现安装锁时先确认是否有其他安装正在进行。

## 4. 卸载并恢复原 TiUP

在解压目录运行，使用安装时相同的 prefix：

```bash
sh ./install.sh --prefix /usr/local --uninstall
```

如果需要管理员权限：

```bash
sudo sh ./install.sh --prefix /usr/local --uninstall
```

有原 TiUP 备份时，脚本会恢复备份；没有备份时，只移除本次创建的命令链接。
如果命令链接已经被其他程序修改，脚本拒绝卸载，避免覆盖后来的改动。
版本目录和用户配置都会保留，脚本不会清理采集数据。

安装过程如果在备份后失败，会尝试恢复原命令。
若出现 `Restore backup manually`，按错误信息中的路径处理，先不要删除备份。
