# pixia-panel

<div align="center">

[![Go](https://img.shields.io/github/go-mod/go-version/pixia1234/pixia-panel?label=Go&color=00ADD8)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18.3+-61DAFB.svg)](https://react.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)
[![Docker Publish](https://github.com/pixia1234/pixia-panel/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/pixia1234/pixia-panel/actions/workflows/docker-publish.yml)
[![Dev CI](https://github.com/pixia1234/pixia-panel/actions/workflows/ci-dev.yml/badge.svg?branch=dev)](https://github.com/pixia1234/pixia-panel/actions/workflows/ci-dev.yml)
[![Release](https://img.shields.io/github/v/release/pixia1234/pixia-panel?display_name=tag)](https://github.com/pixia1234/pixia-panel/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/pixia1234/pixia-panel)](https://goreportcard.com/report/github.com/pixia1234/pixia-panel)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)

</div>

基于Gost + Go + SQLite 的转发面板。

## 部署流程

### Docker Compose 部署

#### 快速部署

面板端（稳定版）：

```
curl -L https://raw.githubusercontent.com/pixia1234/pixia-panel/refs/heads/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh && ./panel_install.sh
```

节点端（稳定版）：

```
curl -L https://raw.githubusercontent.com/pixia1234/pixia-panel/refs/heads/main/node_install.sh -o node_install.sh && chmod +x node_install.sh && ./node_install.sh
```
⚠️在公网环境部署节点时，请与面板用**https**通信，否则是明文传输。

## 升级方法

### 一、面板升级（推荐）

建议每次升级前先重新下载最新版脚本：

```bash
curl -L https://raw.githubusercontent.com/pixia1234/pixia-panel/refs/heads/main/panel_install.sh -o panel_install.sh && chmod +x panel_install.sh
```

执行后选择菜单 **2. 更新面板**：

```bash
./panel_install.sh
```

升级脚本会自动完成：

- 下载对应版本的 `docker-compose` 文件；
- 拉取最新镜像并重建容器；
- 保留现有数据卷（含 `pixia.db`）。

### 二、升级前备份（强烈建议）

同样运行脚本并选择 **4. 导出备份**：

```bash
./panel_install.sh
```

会在当前目录导出 `pixia-backup-*.db`。

### 三、节点升级

在每个节点机器上执行：

```bash
curl -L https://raw.githubusercontent.com/pixia1234/pixia-panel/refs/heads/main/node_install.sh -o node_install.sh && chmod +x node_install.sh
./node_install.sh
```

选择菜单 **2. 更新** 即可完成节点 gost 二进制升级。

或在面板可以直接获取一键安装脚本 运行即可。

## 默认管理员账号

账号: admin_user  
密码: admin_user  
⚠️ 首次登录后请立即修改默认密码！

## 参考与致谢

本项目的部署与交互流程参考原版 flux-panel：https://github.com/bqlpfy/flux-panel。
在此基础上仅进行了性能优化：将面板内存占用由 600MB 降至约 20MB；同时对 gost 进行了裁剪，使其内存占用进一步减少约一半。

## 免责声明

本项目仅供个人学习与研究使用，基于开源项目进行二次开发。

使用本项目所带来的任何风险均由使用者自行承担，包括但不限于：

- 配置不当或使用错误导致的服务异常或不可用；
- 使用本项目引发的网络攻击、封禁、滥用等行为；
- 服务器因使用本项目被入侵、渗透、滥用导致的数据泄露、资源消耗或损失；
- 因违反当地法律法规所产生的任何法律责任。

本项目为开源的流量转发工具，仅限合法、合规用途。使用者必须确保其使用行为符合所在国家或地区的法律法规。

作者不对因使用本项目导致的任何法律责任、经济损失或其他后果承担责任。禁止将本项目用于任何违法或未经授权的行为，包括但不限于网络攻击、数据窃取、非法访问等。

如不同意上述条款，请立即停止使用本项目。

作者对因使用本项目所造成的任何直接或间接损失概不负责，亦不提供任何形式的担保、承诺或技术支持。

请务必在合法、合规、安全的前提下使用本项目。
