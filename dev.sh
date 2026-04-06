#!/bin/bash
# ============================================================
# YMLink-Q2 精简开发脚本 (仅 Go + Vue, 源码直接运行)
#
# 前提: MongoDB 和 InfluxDB Docker 容器已在运行
# 用法: chmod +x dev.sh && ./dev.sh
# 停止: Ctrl+C (自动还原前端补丁)
# ============================================================

set -e

# ============ 颜色 ============
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "  ${GREEN}✓${NC} $1"; }
log_warn()  { echo -e "  ${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "  ${RED}✗${NC} $1"; }

# ============ 路径 ============
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_DIR="$SCRIPT_DIR"
SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"
WEB_DIR="$(dirname "$ENV_DIR")/ymlink-q2-master/.web"
PATCH_DIR="$ENV_DIR/patches"
WEB_PATCH_DIR="$PATCH_DIR/.web"
BACKUP_DIR="$ENV_DIR/.backup_web"

PIDS=()
PATCHED_FILES=()

# ============ 清理 (Ctrl+C) ============
cleanup() {
    echo ""
    echo -e "${YELLOW}正在停止...${NC}"

    # 还原前端补丁
    if [ ${#PATCHED_FILES[@]} -gt 0 ]; then
        for f in "${PATCHED_FILES[@]}"; do
            backup_f="$BACKUP_DIR/$f"
            target_f="$WEB_DIR/$f"
            if [ -f "$backup_f" ]; then
                cp "$backup_f" "$target_f"
            fi
        done
        rm -rf "$BACKUP_DIR"
        log_info "前端补丁已还原 (${#PATCHED_FILES[@]} 个文件)"
    fi

    # 杀进程
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done

    # 杀占用端口的进程
    lsof -ti :8080 2>/dev/null | xargs kill -9 2>/dev/null || true
    lsof -ti :3000 2>/dev/null | xargs kill -9 2>/dev/null || true

    echo -e "${GREEN}已停止${NC}"
    exit 0
}
trap cleanup EXIT INT TERM

# ============ 杀占用端口的旧进程 ============
for port in 8080 3000; do
    old_pid=$(lsof -ti :$port 2>/dev/null || true)
    if [ -n "$old_pid" ]; then
        echo "$old_pid" | xargs kill -9 2>/dev/null || true
        log_warn "端口 $port 旧进程已清理"
        sleep 1
    fi
done

# ============ 准备运行时目录 ============
cd "$SRC_DIR"
for d in "data/friendb" "data/ip2region" "log" "file/task" "file/login" "file/material" "file/message" "file/usedb" "file/qzonedb" "file/materialdb" "file/realinfodb" "apps/server/file" "file/android_pack" "file/ios_pack" "file/ini_pack"; do
    [ ! -d "$d" ] && mkdir -p "$d"
done

# friendb
[ ! -f "$SRC_DIR/data/friendb/friends.friendb" ] && touch "$SRC_DIR/data/friendb/friends.friendb"

# ip2region.xdb
[ ! -f "$SRC_DIR/data/ip2region/ip2region.xdb" ] && [ -f "$ENV_DIR/ip2region.xdb" ] && cp "$ENV_DIR/ip2region.xdb" "$SRC_DIR/data/ip2region/ip2region.xdb"

# ============ 生成 overlay.json ============
OVERLAY_JSON="$PATCH_DIR/overlay.json"
cat > "$OVERLAY_JSON" << EOFOVERLAY
{
  "Replace": {
    "$SRC_DIR/apps/server/main.go": "$PATCH_DIR/apps/server/main.go",
    "$SRC_DIR/plugin/plugin.http.server.go": "$PATCH_DIR/plugin/plugin.http.server.go"
  }
}
EOFOVERLAY

# ============ 前端补丁 (覆盖前先备份) ============
apply_web_patches() {
    if [ -d "$WEB_PATCH_DIR" ] && [ -d "$WEB_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        cd "$WEB_PATCH_DIR"
        for patch_file in $(find . -type f); do
            rel="${patch_file#./}"
            target="$WEB_DIR/$rel"
            if [ -f "$target" ]; then
                mkdir -p "$BACKUP_DIR/$(dirname "$rel")"
                cp "$target" "$BACKUP_DIR/$rel"
            fi
            mkdir -p "$(dirname "$target")"
            cp "$WEB_PATCH_DIR/$rel" "$target"
            PATCHED_FILES+=("$rel")
            log_info "前端补丁: $rel"
        done
    fi
}

# ============ 环境变量 ============
export SYSTEM_MODE=debug
export SYSTEM_LOG_LEVEL=debug
export SYSTEM_MEM_LIMIT=0
export SYSTEM_CPU_LIMIT=4
export SYSTEM_ROBOT_THREAD_LIMIT=1
export SYSTEM_ROBOT_MESSAGE_LIMIT=100
export SYSTEM_DBLOAD=true

export HTTP_SERVER1_URL=:8080
export HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret
export HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./apps/server/html

export MONGO1_URL='localhost:27017/ymlink_q2?authSource=admin'
export MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME=admin
export MONGO1_PASSWORD=admin

export INFLUX1_URL=http://localhost:8086
export INFLUX1_ORG=ymlink
export INFLUX1_TOKEN=ymlink-influx-token

export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
export FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb

export DRIVE_MAPPING='server1#iOSQQ#8.9.80#http://8.130.31.166:8098'
export DRIVE_TIMEOUT=30
export DRIVE_MAX_CONN=50
export MOBILE_PORT=16100

export GOPROXY=https://goproxy.cn,direct

# ============ 启动 Go 后端 ============
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   YMLink-Q2 开发模式 (Go + Vue)                 ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

log_info "启动 Go 后端 (go run -overlay)..."
cd "$SRC_DIR"
go run -overlay="$OVERLAY_JSON" ./apps/server/ &
BACKEND_PID=$!
PIDS+=($BACKEND_PID)

# 等后端
echo -n "  等待后端就绪"
for i in $(seq 1 60); do
    if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo ""; log_error "Go 后端启动失败！查看: cat $SRC_DIR/log/ymlink-q2.log"; exit 1
    fi
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ 2>/dev/null | grep -qE "200|404|401|302"; then
        echo ""; log_info "Go 后端就绪 ✓ (localhost:8080)"; break
    fi
    echo -n "."; sleep 2
done

# ============ 启动 Vue 前端 ============
if command -v node &>/dev/null && [ -d "$WEB_DIR" ]; then
    apply_web_patches
    cd "$WEB_DIR"
    [ ! -d "node_modules" ] && npm install
    log_info "启动 Vue 前端 (Vite 热更新)..."
    npm run dev &
    FRONTEND_PID=$!
    PIDS+=($FRONTEND_PID)
else
    log_warn "未安装 Node.js 或无 .web 目录, 跳过前端"
fi

# ============ 完成 ============
echo ""
echo -e "  ${CYAN}访问地址:${NC}"
echo -e "    ${CYAN}http://localhost:8080/?url=localhost:8080${NC}"
echo -e "    ${CYAN}http://localhost:8080/join-group?url=localhost:8080${NC}  ← 申请入群"
echo ""
echo -e "  ${CYAN}账号:${NC} admin  ${CYAN}密码:${NC} a12345677"
echo ""
echo -e "  ${YELLOW}Ctrl+C 停止${NC}"
echo ""

wait
