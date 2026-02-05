# pixia-panel

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

## 默认管理员账号

账号: admin_user  
密码: admin_user  
⚠️ 首次登录后请立即修改默认密码！


## 构建节点 gost（发布用）

```
cd go-gost
./build.sh
```

输出文件：
- `go-gost/dist/gost-amd64`
- `go-gost/dist/gost-arm64`

发布到 GitHub Release 时，请保持文件名为 `gost-amd64` / `gost-arm64`。

## 参考与致谢

本项目部署与交互流程参考了原版 flux-panel：https://github.com/bqlpfy/flux-panel

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
