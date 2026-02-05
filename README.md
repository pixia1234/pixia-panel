# pixia-panel

Go + SQLite 重写版flux面板。

## 一键安装（面板）

```
curl -fsSL https://raw.githubusercontent.com/pixia1234/pixia-panel/main/panel_install.sh -o panel_install.sh
bash panel_install.sh
```

## 一键安装（节点）

```
curl -fsSL https://raw.githubusercontent.com/pixia1234/pixia-panel/main/node_install.sh -o node_install.sh
bash node_install.sh -a 面板IP:6365 -s 节点密钥
```

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
