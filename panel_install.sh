#!/bin/bash
set -e

export LANG=en_US.UTF-8
export LC_ALL=C

PANEL_VERSION="0.2.0"
REPO="pixia1234/pixia-panel"
BASE_URL="https://github.com/${REPO}/releases/download/${PANEL_VERSION}"
DOCKER_COMPOSEV4_URL="${BASE_URL}/docker-compose-v4.yml"
DOCKER_COMPOSEV6_URL="${BASE_URL}/docker-compose-v6.yml"

COUNTRY=$(curl -s https://ipinfo.io/country || true)
if [ "$COUNTRY" = "CN" ]; then
  DOCKER_COMPOSEV4_URL="https://ghfast.top/${DOCKER_COMPOSEV4_URL}"
  DOCKER_COMPOSEV6_URL="https://ghfast.top/${DOCKER_COMPOSEV6_URL}"
fi

check_docker() {
  if command -v docker-compose &> /dev/null; then
    DOCKER_CMD="docker-compose"
  elif command -v docker &> /dev/null; then
    if docker compose version &> /dev/null; then
      DOCKER_CMD="docker compose"
    else
      echo "错误：检测到 docker，但不支持 'docker compose' 命令。请安装 docker-compose 或更新 docker 版本。"
      exit 1
    fi
  else
    echo "错误：未检测到 docker 或 docker-compose 命令。请先安装 Docker。"
    exit 1
  fi
  echo "检测到 Docker 命令：$DOCKER_CMD"
}

check_ipv6_support() {
  if ip -6 addr show | grep -v "scope link" | grep -q "inet6"; then
    return 0
  elif ifconfig 2>/dev/null | grep -v "fe80:" | grep -q "inet6"; then
    return 0
  else
    return 1
  fi
}

configure_docker_ipv6() {
  OS_TYPE=$(uname -s)
  if [[ "$OS_TYPE" == "Darwin" ]]; then
    echo "✅ macOS Docker Desktop 默认支持 IPv6"
    return 0
  fi

  DOCKER_CONFIG="/etc/docker/daemon.json"
  if [[ $EUID -ne 0 ]]; then
    SUDO_CMD="sudo"
  else
    SUDO_CMD=""
  fi

  if [ -f "$DOCKER_CONFIG" ]; then
    if grep -q '"ipv6"' "$DOCKER_CONFIG"; then
      echo "✅ Docker 已配置 IPv6 支持"
    else
      echo "📝 更新 Docker 配置以启用 IPv6..."
      $SUDO_CMD cp "$DOCKER_CONFIG" "${DOCKER_CONFIG}.backup"
      if command -v jq &> /dev/null; then
        $SUDO_CMD jq '. + {"ipv6": true, "fixed-cidr-v6": "fd00::/80"}' "$DOCKER_CONFIG" > /tmp/daemon.json && $SUDO_CMD mv /tmp/daemon.json "$DOCKER_CONFIG"
      else
        $SUDO_CMD sed -i 's/^{$/{\n  "ipv6": true,\n  "fixed-cidr-v6": "fd00::\/80",/' "$DOCKER_CONFIG"
      fi

      if command -v systemctl &> /dev/null; then
        $SUDO_CMD systemctl restart docker
      elif command -v service &> /dev/null; then
        $SUDO_CMD service docker restart
      fi
      sleep 5
    fi
  else
    echo "📝 创建 Docker 配置文件..."
    $SUDO_CMD mkdir -p /etc/docker
    echo '{
  "ipv6": true,
  "fixed-cidr-v6": "fd00::/80"
}' | $SUDO_CMD tee "$DOCKER_CONFIG" > /dev/null

    if command -v systemctl &> /dev/null; then
      $SUDO_CMD systemctl restart docker
    elif command -v service &> /dev/null; then
      $SUDO_CMD service docker restart
    fi
    sleep 5
  fi
}

show_menu() {
  echo "==============================================="
  echo "          面板管理脚本"
  echo "==============================================="
  echo "请选择操作："
  echo "1. 安装面板"
  echo "2. 更新面板"
  echo "3. 卸载面板"
  echo "4. 导出备份"
  echo "5. 退出"
  echo "==============================================="
}

generate_random() {
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c16
}

delete_self() {
  echo ""
  echo "🗑️ 操作已完成，正在清理脚本文件..."
  SCRIPT_PATH="$(readlink -f "$0" 2>/dev/null || realpath "$0" 2>/dev/null || echo "$0")"
  sleep 1
  rm -f "$SCRIPT_PATH" && echo "✅ 脚本文件已删除" || echo "❌ 删除脚本文件失败"
}

get_config_params() {
  echo "🔧 请输入配置参数："

  read -p "前端端口（默认 6366）: " FRONTEND_PORT
  FRONTEND_PORT=${FRONTEND_PORT:-6366}

  read -p "后端端口（默认 6365）: " BACKEND_PORT
  BACKEND_PORT=${BACKEND_PORT:-6365}

  read -p "JWT 密钥（默认随机生成）: " JWT_SECRET
  JWT_SECRET=${JWT_SECRET:-$(generate_random)}

  DEFAULT_BACKEND_IMAGE="pixia1234/pixia-panel-backend:latest"
  DEFAULT_FRONTEND_IMAGE="pixia1234/pixia-panel-frontend:latest"

  PIXIA_BACKEND_IMAGE=${PIXIA_BACKEND_IMAGE:-$DEFAULT_BACKEND_IMAGE}
  PIXIA_FRONTEND_IMAGE=${PIXIA_FRONTEND_IMAGE:-$DEFAULT_FRONTEND_IMAGE}
}

fetch_compose_file() {
  local url
  local target
  target="docker-compose.yml"

  if check_ipv6_support; then
    url="$DOCKER_COMPOSEV6_URL"
    echo "📡 选择配置文件：docker-compose-v6.yml"
  else
    url="$DOCKER_COMPOSEV4_URL"
    echo "📡 选择配置文件：docker-compose-v4.yml"
  fi

  if [ -f "./docker-compose-v4.yml" ] || [ -f "./docker-compose-v6.yml" ]; then
    if check_ipv6_support; then
      cp ./docker-compose-v6.yml "$target"
    else
      cp ./docker-compose-v4.yml "$target"
    fi
    return 0
  fi

  curl -L -o "$target" "$url"
}

install_panel() {
  echo "🚀 开始安装面板..."
  check_docker
  get_config_params

  echo "🔽 准备配置文件..."
  fetch_compose_file

  if check_ipv6_support; then
    echo "🚀 系统支持 IPv6，自动启用 IPv6 配置..."
    configure_docker_ipv6
  fi

  cat > .env <<ENVEOF
FRONTEND_PORT=$FRONTEND_PORT
BACKEND_PORT=$BACKEND_PORT
JWT_SECRET=$JWT_SECRET
PIXIA_BACKEND_IMAGE=$PIXIA_BACKEND_IMAGE
PIXIA_FRONTEND_IMAGE=$PIXIA_FRONTEND_IMAGE
ENVEOF

  echo "⬇️ 拉取镜像..."
  $DOCKER_CMD pull

  echo "🚀 启动 docker 服务..."
  $DOCKER_CMD up -d

  echo "🎉 部署完成"
  echo "🌐 访问地址: http://服务器IP:$FRONTEND_PORT"
  echo "💡 默认管理员账号: admin_user / admin_user"
  echo "⚠️  登录后请立即修改默认密码！"
}

update_panel() {
  echo "🔄 开始更新面板..."
  check_docker

  echo "🔽 下载最新配置文件..."
  fetch_compose_file

  if check_ipv6_support; then
    echo "🚀 系统支持 IPv6，自动启用 IPv6 配置..."
    configure_docker_ipv6
  fi

  echo "🛑 停止当前服务..."
  $DOCKER_CMD down

  echo "⬇️ 拉取最新镜像..."
  $DOCKER_CMD pull

  echo "🚀 启动更新后的服务..."
  $DOCKER_CMD up -d

  echo "✅ 更新完成"
}

uninstall_panel() {
  echo "🗑️ 开始卸载面板..."

  read -p "确认卸载面板吗？此操作将删除容器与数据卷 (y/N): " confirm
  if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "❌ 取消卸载"
    return 0
  fi

  check_docker
  $DOCKER_CMD down -v
  echo "✅ 卸载完成"
}

backup_panel() {
  echo "📦 导出数据备份..."
  check_docker

  BACKUP_FILE="pixia-backup-$(date +%Y%m%d%H%M%S).db"
  docker run --rm -v pixia_data:/data -v "$(pwd)":/backup alpine sh -c "cp /data/pixia.db /backup/${BACKUP_FILE}" 2>/dev/null || true

  if [[ -f "$BACKUP_FILE" ]]; then
    echo "✅ 备份完成：$BACKUP_FILE"
  else
    echo "❌ 备份失败，请确认 pixia_data 卷是否存在"
  fi
}

main() {
  while true; do
    show_menu
    read -p "请输入选项 (1-5): " choice

    case $choice in
      1)
        install_panel
        delete_self
        exit 0
        ;;
      2)
        update_panel
        delete_self
        exit 0
        ;;
      3)
        uninstall_panel
        delete_self
        exit 0
        ;;
      4)
        backup_panel
        delete_self
        exit 0
        ;;
      5)
        echo "👋 退出脚本"
        delete_self
        exit 0
        ;;
      *)
        echo "❌ 无效选项，请输入 1-5"
        echo ""
        ;;
    esac
  done
}

main
