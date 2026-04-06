#!/bin/bash
# ============================================================
# YMLink-Q2 一键 Docker 全容器部署
#
# 核心原则: ymlink-q2-new-master/ 目录代码完全不动！
#   Go 后端: Docker 内 go build -overlay (补丁在 Dockerfile 里处理)
#   MongoDB + InfluxDB: 自动检测镜像，不存在则拉取
#
# 用法:
#   ./one-click-docker.sh          # 编译+启动
#   ./one-click-docker.sh rebuild  # 强制重新编译
#   ./one-click-docker.sh stop     # 停止
#   ./one-click-docker.sh logs     # 看日志
#   ./one-click-docker.sh status   # 看状态
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_DIR="$SCRIPT_DIR"
SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"
COMPOSE_FILE="$ENV_DIR/docker-compose.yml"

# ============ 自动检测 Docker 路径 (Mac Docker Desktop) ============
if ! command -v docker &>/dev/null; then
    DOCKER_PATHS=(
        "/Applications/Docker.app/Contents/Resources/bin"
        "/usr/local/bin"
        "$HOME/.docker/bin"
        "/opt/homebrew/bin"
    )
    for dp in "${DOCKER_PATHS[@]}"; do
        if [ -x "$dp/docker" ]; then
            export PATH="$dp:$PATH"
            break
        fi
    done
fi

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

ACTION="${1:-up}"

case "$ACTION" in
    stop)   docker compose -f "$COMPOSE_FILE" down 2>/dev/null; log_info "已停止 (数据卷保留)"; exit 0;;
    logs)   docker compose -f "$COMPOSE_FILE" logs -f --tail=100; exit 0;;
    status) docker compose -f "$COMPOSE_FILE" ps; exit 0;;
    rebuild) REBUILD="--build --no-cache";;
    up|"")   REBUILD="--build";;
    *)       echo "用法: $0 [up|rebuild|stop|logs|status]"; exit 1;;
esac

ensure_docker_image() {
    docker image inspect "$1" &>/dev/null && log_info "镜像: $1 ✓" && return 0
    log_warn "拉取: $1 ..."
    docker pull "$1" || { log_error "拉取失败！请设置 registry-mirrors"; exit 1; }
    log_info "拉取成功: $1"
}
ensure_docker_running() {
    docker info &>/dev/null 2>&1 && return 0
    log_warn "启动 Docker Desktop..."; open -a Docker
    for i in $(seq 1 60); do docker info &>/dev/null 2>&1 && { log_info "Docker 就绪"; return 0; }; sleep 2; done
    log_error "Docker 超时"; exit 1
}

# ============================================================
# [1] 检查环境
# ============================================================
log_step "1/5  检查环境"

[ ! -d "$SRC_DIR" ] && { log_error "源码目录不存在: $SRC_DIR"; exit 1; }
[ ! -f "$SRC_DIR/go.mod" ] && { log_error "源码不完整"; exit 1; }
log_info "源码: $SRC_DIR"

command -v docker &>/dev/null || { log_error "未装 Docker"; exit 1; }
log_info "Docker: $(docker --version | awk '{print $3}' | tr -d ',')"
ensure_docker_running
docker compose version &>/dev/null || { log_error "docker compose 不可用"; exit 1; }
log_info "Compose: $(docker compose version --short)"

# ============================================================
# [2] 检查必要文件
# ============================================================
log_step "2/5  检查必要文件"

E=0
[ ! -f "$ENV_DIR/ip2region.xdb" ] && { log_error "缺 ip2region.xdb"; E=$((E+1)); }
[ ! -f "$ENV_DIR/patches/apps/server/main.go" ] && { log_error "缺 Go 补丁 main.go"; E=$((E+1)); }
[ ! -f "$ENV_DIR/patches/plugin/plugin.http.server.go" ] && { log_error "缺 Go 补丁 http.server.go"; E=$((E+1)); }
[ ! -f "$COMPOSE_FILE" ] && { log_error "缺 docker-compose.yml"; E=$((E+1)); }
[ $E -gt 0 ] && exit 1

# 确保 html/ 和 Dockerfile COPY 的目录都存在
mkdir -p "$SRC_DIR/apps/server/html"
log_info "全部文件就绪 ✓"

# ============================================================
# [3] 预拉取 Docker 镜像 (自动检测)
# ============================================================
log_step "3/5  检测/拉取 Docker 镜像"

ensure_docker_image "mongo:6.0"
ensure_docker_image "influxdb:2.7"
ensure_docker_image "golang:1.22.4-alpine"
ensure_docker_image "alpine:3.19"
log_info "所有镜像就绪 ✓"

# ============================================================
# [4] 编译+启动
# ============================================================
log_step "4/5  编译+启动容器"

[ "$ACTION" = "rebuild" ] && log_info "强制重新编译 (--no-cache)"
cd "$ENV_DIR"
docker compose -f "$COMPOSE_FILE" up -d $REBUILD

# ============================================================
# [5] 验证
# ============================================================
log_step "5/5  验证服务"

log_info "等待 5 秒..."
sleep 5

echo ""
docker compose -f "$COMPOSE_FILE" ps
echo ""

ERR=0
docker ps --format '{{.Names}}' | grep -q "ymlink-mongo"  && log_info "MongoDB ✓"       || { log_error "MongoDB ✗"; ERR=$((ERR+1)); }
docker ps --format '{{.Names}}' | grep -q "ymlink-influx"  && log_info "InfluxDB ✓"      || { log_error "InfluxDB ✗"; ERR=$((ERR+1)); }
docker ps --format '{{.Names}}' | grep -q "ymlink-server" && log_info "YMLink Server ✓" || { log_error "YMLink Server ✗"; ERR=$((ERR+1)); }

echo ""
if [ $ERR -eq 0 ]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║  YMLink-Q2 Docker 部署成功                      ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
else
    echo -e "${YELLOW}部分服务未就绪: docker logs <容器名>${NC}"
fi
echo ""
echo -e "  ${CYAN}http://localhost:8080${NC}  (API + 前端)"
echo -e "  ${CYAN}http://localhost:8086${NC}  (InfluxDB)"
echo ""
echo -e "  $0 logs     查看日志"
echo -e "  $0 stop     停止"
echo -e "  $0 rebuild  重编译"
echo ""
