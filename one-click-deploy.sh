#!/bin/bash
# ============================================================
# YMLink-Q2 一键编译部署 (Mac 本地, go build 编译后运行)
#
# 核心原则: ymlink-q2-new-master/ 目录代码完全不动！
#   Go 后端补丁:  go build -overlay (编译时替换，源码不变)
#   Vue 前端补丁: 覆盖 → 编译 → 还原 (源码恢复原样)
#
# 用法: chmod +x one-click-deploy.sh && ./one-click-deploy.sh
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_DIR="$SCRIPT_DIR"
SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"
PATCH_DIR="$ENV_DIR/patches"
WEB_DIR="$SRC_DIR/.web"
WEB_PATCH_DIR="$PATCH_DIR/.web"
BACKUP_DIR="$ENV_DIR/.backup_web"
BINARY="$SRC_DIR/ymlink-server"

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

# ---- 前端补丁 ----
PATCHED_FILES=()
apply_web_patches() {
    [ ! -d "$WEB_PATCH_DIR" ] || [ ! -d "$WEB_DIR" ] && return 0
    mkdir -p "$BACKUP_DIR"
    local count=0
    while IFS= read -r pf; do
        local rel="${pf#$WEB_PATCH_DIR/}"
        local orig="$WEB_DIR/$rel" bak="$BACKUP_DIR/$rel"
        [ -f "$orig" ] && { mkdir -p "$(dirname "$bak")"; cp -p "$orig" "$bak"; }
        mkdir -p "$(dirname "$orig")"; cp "$pf" "$orig"
        PATCHED_FILES+=("$rel"); count=$((count+1))
        log_info "  补丁: .web/$rel"
    done < <(find "$WEB_PATCH_DIR" -type f ! -name '.DS_Store')
    [ $count -gt 0 ] && log_info "已应用 $count 个前端补丁"
}
restore_web_patches() {
    [ ${#PATCHED_FILES[@]} -eq 0 ] && return 0
    log_info "还原前端补丁..."
    for rel in "${PATCHED_FILES[@]}"; do
        local orig="$WEB_DIR/$rel" bak="$BACKUP_DIR/$rel"
        [ -f "$bak" ] && cp -p "$bak" "$orig" && log_info "  还原: .web/$rel" || { rm -f "$orig"; log_info "  移除: .web/$rel"; }
    done
    rm -rf "$BACKUP_DIR"
    log_info "前端源码已还原 ✓"
}

# ---- 进程管理 ----
cleanup() {
    echo ""; log_warn "停止服务..."
    [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null && { kill "$SERVER_PID"; wait "$SERVER_PID" 2>/dev/null || true; }
    log_info "服务已停止"
    exit 0
}
trap cleanup SIGINT SIGTERM

# ---- Docker 通用 ----
ensure_docker_image() {
    docker image inspect "$1" &>/dev/null && log_info "镜像: $1 ✓" && return 0
    log_warn "拉取: $1 ..."
    docker pull "$1" || { log_error "拉取失败！请设置 Docker 镜像加速"; exit 1; }
    log_info "拉取成功: $1"
}
ensure_docker_running() {
    docker info &>/dev/null 2>&1 && return 0
    log_warn "启动 Docker Desktop..."; open -a Docker
    for i in $(seq 1 60); do docker info &>/dev/null 2>&1 && { log_info "Docker 已就绪"; return 0; }; sleep 2; done
    log_error "Docker 启动超时"; exit 1
}
ensure_mongo() {
    local C="ymlink-mongo" IMG="mongo:6.0"
    local EXISTING=$(get_container_on_port 27017)
    if [ -n "$EXISTING" ]; then
        log_info "MongoDB 已运行 (容器: $EXISTING, 保持不变)"; MONGO_CONTAINER="$EXISTING"; return 0
    fi
    if docker ps -a --format '{{.Names}}' | grep -q "^${C}$"; then
        docker start "$C" 2>/dev/null
    else
        ensure_docker_image "$IMG"
        docker run -d --name "$C" -p 27017:27017 -v ymlink-mongo-data:/data/db \
            -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=admin "$IMG" --auth
    fi
    MONGO_CONTAINER="$C"
    echo -n "  等待 MongoDB"; for i in $(seq 1 40); do
        docker exec "$MONGO_CONTAINER" mongosh --quiet --eval "db.runCommand({ping:1})" -u admin -p admin --authenticationDatabase admin &>/dev/null && { echo ""; log_info "MongoDB 就绪"; return 0; }
        echo -n "."; sleep 1; done
    echo ""; log_error "MongoDB 超时"; exit 1
}
ensure_influx() {
    local C="ymlink-influx" IMG="influxdb:2.7"
    local EXISTING=$(get_container_on_port 8086)
    if [ -n "$EXISTING" ]; then
        log_info "InfluxDB 已运行 (容器: $EXISTING, 保持不变)"; INFLUX_CONTAINER="$EXISTING"; return 0
    fi
    if docker ps -a --format '{{.Names}}' | grep -q "^${C}$"; then
        docker start "$C" 2>/dev/null
    else
        ensure_docker_image "$IMG"
        docker run -d --name "$C" -p 8086:8086 -v ymlink-influx-data:/var/lib/influxdb2 \
            -e DOCKER_INFLUXDB_INIT_MODE=setup -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=admin12345678 -e DOCKER_INFLUXDB_INIT_ORG=ymlink \
            -e DOCKER_INFLUXDB_INIT_BUCKET=realtime -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=ymlink-influx-token \
            -e DOCKER_INFLUXDB_INIT_RETENTION=168h "$IMG"
    fi
    INFLUX_CONTAINER="$C"
    echo -n "  等待 InfluxDB"; for i in $(seq 1 40); do
        curl -s http://localhost:8086/health 2>/dev/null | grep -q '"status":"pass"' && { echo ""; log_info "InfluxDB 就绪"; return 0; }
        echo -n "."; sleep 1; done
    echo ""; log_error "InfluxDB 超时"; exit 1
}
check_port() {
    local PORT=$1 SERVICE=$2
    if lsof -i :"$PORT" -sTCP:LISTEN &>/dev/null; then
        local PIDS_ON_PORT=$(lsof -i :"$PORT" -sTCP:LISTEN -t 2>/dev/null)
        for PROC in $PIDS_ON_PORT; do
            local PNAME=$(ps -p "$PROC" -o comm= 2>/dev/null || echo "?")
            echo "$PNAME" | grep -qiE "docker|vpnkit|com.docker" && continue
            log_warn "端口 $PORT ($SERVICE) 被 $PNAME (PID:$PROC) 占用，自动终止..."
            kill "$PROC" 2>/dev/null || true; sleep 1
            kill -0 "$PROC" 2>/dev/null && { kill -9 "$PROC" 2>/dev/null || true; sleep 1; }
            log_info "已终止 $PNAME，端口 $PORT 已释放"
        done
    fi; return 0
}

get_container_on_port() {
    local PORT=$1
    docker ps --format '{{.Names}}\t{{.Ports}}' 2>/dev/null | grep "0.0.0.0:${PORT}->" | awk -F'\t' '{print $1}' | head -1
}

MONGO_CONTAINER="ymlink-mongo"
INFLUX_CONTAINER="ymlink-influx"

# ============================================================
# [1] 检查依赖
# ============================================================
log_step "1/8  检查依赖"
E=0
[ ! -d "$SRC_DIR" ] && { log_error "源码目录不存在: $SRC_DIR"; exit 1; }
[ ! -f "$SRC_DIR/go.mod" ] && { log_error "源码不完整"; exit 1; }
log_info "源码: $SRC_DIR"
command -v go &>/dev/null && log_info "Go: $(go version | awk '{print $3}')" || { log_error "未装 Go"; E=$((E+1)); }
command -v docker &>/dev/null && log_info "Docker: $(docker --version | awk '{print $3}' | tr -d ',')" || { log_error "未装 Docker"; E=$((E+1)); }
[ -f "$PATCH_DIR/apps/server/main.go" ] && [ -f "$PATCH_DIR/plugin/plugin.http.server.go" ] && log_info "Go 补丁就绪" || { log_error "缺 Go 补丁"; E=$((E+1)); }
[ -f "$ENV_DIR/ip2region.xdb" ] || [ -f "$SRC_DIR/data/ip2region/ip2region.xdb" ] && log_info "ip2region.xdb 就绪" || { log_error "缺 ip2region.xdb"; E=$((E+1)); }
[ $E -gt 0 ] && exit 1

# ============================================================
# [2] 检查端口
# ============================================================
log_step "2/8  检查端口"
check_port 8080 "Go后端"
log_info "端口就绪 ✓"

# ============================================================
# [3] Docker 服务
# ============================================================
log_step "3/8  启动 Docker 服务"
ensure_docker_running; ensure_mongo; ensure_influx

# ============================================================
# [4] 初始化数据库
# ============================================================
log_step "4/8  初始化数据库"
MONGO_INIT_FLAG="$ENV_DIR/.mongo_initialized"
if [ ! -f "$MONGO_INIT_FLAG" ]; then
    docker exec "$MONGO_CONTAINER" mongosh -u admin -p admin --authenticationDatabase admin --quiet --eval '
        db=db.getSiblingDB("ymlink_q2");
        ["account","account_key","drive_record","event","friend","friend_group","friend_label","friend_message","group_member","quest","quest_group","quest_template","robot","robot_group","setting","system_log","task","task_template","user","vps","worker"].forEach(function(n){if(!db.getCollectionNames().includes(n))db.createCollection(n)});
        print("集合: "+db.getCollectionNames().length+" 个");
    ' 2>/dev/null && touch "$MONGO_INIT_FLAG" || true
fi
docker exec "$MONGO_CONTAINER" mongosh --quiet "mongodb://admin:admin@localhost:27017/ymlink_q2?authSource=admin" --eval 'db.getCollectionNames().length' &>/dev/null && log_info "MongoDB 连通 ✓" || { log_error "MongoDB 连通失败"; exit 1; }
INFLUX_INIT_FLAG="$ENV_DIR/.influx_initialized"
[ ! -f "$INFLUX_INIT_FLAG" ] && { docker exec "$INFLUX_CONTAINER" influx bucket create --name history --org ymlink --retention 8760h --token ymlink-influx-token 2>/dev/null || true; touch "$INFLUX_INIT_FLAG"; }

# ============================================================
# [5] 运行时目录 + 数据文件
# ============================================================
log_step "5/8  准备运行时环境"
cd "$SRC_DIR"
for d in data/friendb data/ip2region log file/task file/login file/material file/message file/usedb file/qzonedb file/materialdb file/realinfodb file/android_pack file/ios_pack file/ini_pack apps/server/file; do mkdir -p "$d"; done
[ ! -f "data/ip2region/ip2region.xdb" ] && cp "$ENV_DIR/ip2region.xdb" data/ip2region/ip2region.xdb
[ ! -f "data/friendb/friends.friendb" ] && touch data/friendb/friends.friendb
mkdir -p apps/server/html
log_info "运行时目录和数据文件就绪 ✓"

# ============================================================
# [6] 编译 Go 后端
# ============================================================
log_step "6/8  编译 Go 后端 (go build -overlay)"
cd "$SRC_DIR"; export GOPROXY=https://goproxy.cn,direct
go mod download 2>/dev/null || true
OVERLAY_JSON="$PATCH_DIR/overlay.json"
cat > "$OVERLAY_JSON" << EOF
{"Replace":{"$SRC_DIR/apps/server/main.go":"$PATCH_DIR/apps/server/main.go","$SRC_DIR/plugin/plugin.http.server.go":"$PATCH_DIR/plugin/plugin.http.server.go"}}
EOF
log_info "编译中..."
go build -overlay="$OVERLAY_JSON" -o "$BINARY" ./apps/server/
log_info "编译成功: ymlink-server ($(du -h "$BINARY" | awk '{print $1}')) ✓"

# ============================================================
# [7] 编译前端 (补丁 → 编译 → 还原)
# ============================================================
log_step "7/8  编译前端"
if [ -d "$WEB_DIR" ] && command -v npm &>/dev/null; then
    apply_web_patches
    cd "$WEB_DIR"
    [ ! -d "node_modules" ] && npm install
    log_info "编译前端 (npm run build)..."
    npm run build
    restore_web_patches  # 编译完立即还原
    if [ -d "$WEB_DIR/dist" ]; then
        mkdir -p "$SRC_DIR/apps/server/html"
        cp -r "$WEB_DIR/dist/"* "$SRC_DIR/apps/server/html/"
        log_info "前端编译产物 → html/ ✓"
    fi
else
    [ -f "$SRC_DIR/apps/server/html/index.html" ] && log_info "使用已有 html/" || log_warn "无前端"
fi

# ============================================================
# [8] 启动
# ============================================================
log_step "8/8  启动服务"
cd "$SRC_DIR"

export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_ROBOT_THREAD_LIMIT=1 SYSTEM_ROBOT_MESSAGE_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret
export HTTP_SERVER1_AUTH_TIMEOUT=604800 HTTP_SERVER1_HTML_PATH=./apps/server/html
export MONGO1_URL='localhost:27017/ymlink_q2?authSource=admin' MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=ymlink INFLUX1_TOKEN=ymlink-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING='server1#iOSQQ#8.9.80#http://8.130.31.166:8098' DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50
export MOBILE_PORT=16100

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  YMLink-Q2 编译部署完成                         ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo -e "  ${CYAN}http://localhost:8080${NC}  (API + 前端)"
echo -e "  ${YELLOW}Ctrl+C 停止${NC}"
echo ""

"$BINARY" &
SERVER_PID=$!
log_info "PID: $SERVER_PID"
wait $SERVER_PID
