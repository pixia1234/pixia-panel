#!/bin/bash

get_architecture() {
  ARCH=$(uname -m)
  case $ARCH in
    x86_64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      echo "amd64"
      ;;
  esac
}

GOST_VERSION="0.1.0"
REPO="pixia1234/pixia-panel"
BASE_URL="https://github.com/${REPO}/releases/download/${GOST_VERSION}"
INSTALL_DIR="/etc/gost"

build_download_url() {
  local ARCH=$(get_architecture)
  echo "${BASE_URL}/gost-${ARCH}"
}

DOWNLOAD_URL=$(build_download_url)
COUNTRY=$(curl -s https://ipinfo.io/country || true)
if [ "$COUNTRY" = "CN" ]; then
  DOWNLOAD_URL="https://ghfast.top/${DOWNLOAD_URL}"
fi

show_menu() {
  echo "==============================================="
  echo "              节点管理脚本"
  echo "==============================================="
  echo "请选择操作："
  echo "1. 安装"
  echo "2. 更新"
  echo "3. 卸载"
  echo "4. 退出"
  echo "==============================================="
}

delete_self() {
  echo ""
  echo "🗑️ 操作已完成，正在清理脚本文件..."
  SCRIPT_PATH="$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")"
  sleep 1
  rm -f "$SCRIPT_PATH" && echo "✅ 脚本文件已删除" || echo "❌ 删除脚本文件失败"
}

get_config_params() {
  if [[ -z "$SERVER_ADDR" || -z "$SECRET" ]]; then
    echo "请输入配置参数："

    if [[ -z "$SERVER_ADDR" ]]; then
      read -p "服务器地址(面板IP:6365): " SERVER_ADDR
    fi

    if [[ -z "$SECRET" ]]; then
      read -p "节点密钥: " SECRET
    fi

    if [[ -z "$SERVER_ADDR" || -z "$SECRET" ]]; then
      echo "❌ 参数不完整，操作取消。"
      exit 1
    fi
  fi
}

while getopts "a:s:" opt; do
  case $opt in
    a) SERVER_ADDR="$OPTARG" ;;
    s) SECRET="$OPTARG" ;;
    *) echo "❌ 无效参数"; exit 1 ;;
  esac
done

install_gost() {
  echo "🚀 开始安装 GOST 节点..."
  get_config_params

  mkdir -p "$INSTALL_DIR"

  if systemctl list-units --full -all | grep -Fq "gost.service"; then
    echo "🔍 检测到已存在的gost服务"
    systemctl stop gost 2>/dev/null || true
    systemctl disable gost 2>/dev/null || true
  fi

  [[ -f "$INSTALL_DIR/gost" ]] && rm -f "$INSTALL_DIR/gost"

  echo "⬇️ 下载 gost..."
  curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/gost"
  if [[ ! -f "$INSTALL_DIR/gost" || ! -s "$INSTALL_DIR/gost" ]]; then
    echo "❌ 下载失败，请检查网络或下载链接。"
    exit 1
  fi
  chmod +x "$INSTALL_DIR/gost"

  echo "🔎 gost 版本：$($INSTALL_DIR/gost -V 2>/dev/null || echo unknown)"

  CONFIG_FILE="$INSTALL_DIR/config.json"
  cat > "$CONFIG_FILE" <<EOF
{
  "addr": "$SERVER_ADDR",
  "secret": "$SECRET",
  "http": 1,
  "tls": 1,
  "socks": 1
}
EOF

  GOST_CONFIG="$INSTALL_DIR/gost.yml"
  cat > "$GOST_CONFIG" <<EOF
services: []
EOF

  chmod 600 "$INSTALL_DIR"/*.json || true

  SERVICE_FILE="/etc/systemd/system/gost.service"
  cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Pixia Gost Node
After=network.target

[Service]
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/gost
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable gost
  systemctl start gost

  echo "🔄 检查服务状态..."
  if systemctl is-active --quiet gost; then
    echo "✅ 安装完成，gost服务已启动并设置为开机启动。"
    echo "📁 配置目录: $INSTALL_DIR"
  else
    echo "❌ gost服务启动失败，请执行以下命令查看日志："
    echo "journalctl -u gost -f"
  fi
}

update_gost() {
  echo "🔄 开始更新 GOST..."
  if [[ ! -d "$INSTALL_DIR" ]]; then
    echo "❌ GOST 未安装，请先选择安装。"
    return 1
  fi

  echo "📥 使用下载地址: $DOWNLOAD_URL"

  echo "⬇️ 下载最新版本..."
  curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/gost.new"
  if [[ ! -f "$INSTALL_DIR/gost.new" || ! -s "$INSTALL_DIR/gost.new" ]]; then
    echo "❌ 下载失败。"
    return 1
  fi

  if systemctl list-units --full -all | grep -Fq "gost.service"; then
    systemctl stop gost
  fi

  mv "$INSTALL_DIR/gost.new" "$INSTALL_DIR/gost"
  chmod +x "$INSTALL_DIR/gost"

  echo "🔎 新版本：$($INSTALL_DIR/gost -V 2>/dev/null || echo unknown)"

  systemctl start gost
  echo "✅ 更新完成，服务已重新启动。"
}

uninstall_gost() {
  echo "🗑️ 开始卸载 GOST..."

  read -p "确认卸载 GOST 吗？此操作将删除所有相关文件 (y/N): " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "❌ 取消卸载"
    return 0
  fi

  if systemctl list-units --full -all | grep -Fq "gost.service"; then
    systemctl stop gost 2>/dev/null || true
    systemctl disable gost 2>/dev/null || true
  fi

  if [[ -f "/etc/systemd/system/gost.service" ]]; then
    rm -f "/etc/systemd/system/gost.service"
  fi

  if [[ -d "$INSTALL_DIR" ]]; then
    rm -rf "$INSTALL_DIR"
  fi

  systemctl daemon-reload
  echo "✅ 卸载完成"
}

main() {
  if [[ -n "$SERVER_ADDR" && -n "$SECRET" ]]; then
    install_gost
    delete_self
    exit 0
  fi

  while true; do
    show_menu
    read -p "请输入选项 (1-4): " choice

    case $choice in
      1)
        install_gost
        delete_self
        exit 0
        ;;
      2)
        update_gost
        delete_self
        exit 0
        ;;
      3)
        uninstall_gost
        delete_self
        exit 0
        ;;
      4)
        echo "👋 退出脚本"
        delete_self
        exit 0
        ;;
      *)
        echo "❌ 无效选项，请输入 1-4"
        echo ""
        ;;
    esac
  done
}

main
