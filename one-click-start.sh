#!/bin/bash
# ============================================================
# YMLink-Q2 一键启动脚本 (Mac 开发环境, go run 不编译)
#
# 目录结构 (平级放置):
#   ymlink-q2-new-master/   ← 从 gitee 拉取的源码 (不修改)
#   ymlink-q2-env/          ← 本环境目录
#     └── patches/
#         ├── apps/server/main.go               ← Go 补丁 (overlay)
#         ├── plugin/plugin.http.server.go       ← Go 补丁 (overlay)
#         └── .web/                              ← Vue 前端补丁
#             └── setting.ts                     ← API地址改为 localhost
#
# 补丁机制:
#   Go 后端: go run -overlay (编译时文件替换，不动源码)
#   Vue 前端: 启动前覆盖 → 停止时自动还原 (保护源码)
#
# 用法: chmod +x one-click-start.sh && ./one-click-start.sh
# 停止: Ctrl+C (自动还原前端补丁)
# ============================================================

set -e

# ============ 路径 ============
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_DIR="$SCRIPT_DIR"
SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"
PATCH_DIR="$ENV_DIR/patches"
WEB_DIR="$SRC_DIR/.web"
WEB_PATCH_DIR="$PATCH_DIR/.web"
BACKUP_DIR="$ENV_DIR/.backup_web"

# Docker 容器名 (会在 ensure_mongo/ensure_influx 中自动检测)
MONGO_CONTAINER="ymlink-mongo"
INFLUX_CONTAINER="ymlink-influx"

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

# ============ 颜色 ============
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# ============ 前端补丁管理 ============
PATCHED_FILES=()  # 记录所有被补丁的文件，用于还原

# 应用前端补丁 (覆盖前先备份)
apply_web_patches() {
    if [ ! -d "$WEB_PATCH_DIR" ] || [ ! -d "$WEB_DIR" ]; then
        return 0
    fi
    mkdir -p "$BACKUP_DIR"
    local count=0
    # 遍历补丁目录的所有文件
    while IFS= read -r patch_file; do
        local rel_path="${patch_file#$WEB_PATCH_DIR/}"
        local orig_file="$WEB_DIR/$rel_path"
        local backup_file="$BACKUP_DIR/$rel_path"
        # 备份原文件 (如果存在)
        if [ -f "$orig_file" ]; then
            mkdir -p "$(dirname "$backup_file")"
            cp -p "$orig_file" "$backup_file"  # -p 保留时间戳
        fi
        # 覆盖
        mkdir -p "$(dirname "$orig_file")"
        cp "$patch_file" "$orig_file"
        PATCHED_FILES+=("$rel_path")
        count=$((count+1))
        log_info "  补丁: .web/$rel_path"
    done < <(find "$WEB_PATCH_DIR" -type f ! -name '.DS_Store')
    if [ $count -gt 0 ]; then
        log_info "已应用 $count 个前端补丁"
    fi
}

# 还原前端补丁 (从备份恢复)
restore_web_patches() {
    if [ ${#PATCHED_FILES[@]} -eq 0 ]; then
        return 0
    fi
    log_info "还原前端补丁..."
    for rel_path in "${PATCHED_FILES[@]}"; do
        local orig_file="$WEB_DIR/$rel_path"
        local backup_file="$BACKUP_DIR/$rel_path"
        if [ -f "$backup_file" ]; then
            cp -p "$backup_file" "$orig_file"
            log_info "  还原: .web/$rel_path"
        else
            # 原本不存在的文件，删除
            rm -f "$orig_file"
            log_info "  移除: .web/$rel_path (补丁新增的文件)"
        fi
    done
    # 清理备份目录
    rm -rf "$BACKUP_DIR"
    log_info "前端补丁已全部还原，源码未被修改"
}

# ============ 进程管理 ============
PIDS=()
cleanup() {
    echo ""
    log_warn "正在停止所有服务..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null
            wait "$pid" 2>/dev/null || true
        fi
    done
    # 还原前端补丁
    restore_web_patches
    log_info "Go/Vue 服务已停止 (Docker 容器继续运行)"
    log_info "停止 Docker: docker stop ymlink-mongo ymlink-influx"
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# ============================================================
# 通用函数
# ============================================================
check_port() {
    local PORT=$1 SERVICE=$2
    if lsof -i :"$PORT" -sTCP:LISTEN &>/dev/null; then
        local PIDS_ON_PORT=$(lsof -i :"$PORT" -sTCP:LISTEN -t 2>/dev/null)
        for PROC in $PIDS_ON_PORT; do
            local PNAME=$(ps -p "$PROC" -o comm= 2>/dev/null || echo "unknown")
            # Docker 占用的端口不管
            if echo "$PNAME" | grep -qiE "docker|vpnkit|com.docker"; then continue; fi
            log_warn "端口 $PORT ($SERVICE) 被 $PNAME (PID:$PROC) 占用，自动终止..."
            kill "$PROC" 2>/dev/null || true
            sleep 1
            # 如果 kill 不掉，强杀
            if kill -0 "$PROC" 2>/dev/null; then
                kill -9 "$PROC" 2>/dev/null || true
                sleep 1
            fi
            log_info "已终止 $PNAME (PID:$PROC)，端口 $PORT 已释放"
        done
    fi
    return 0
}

ensure_docker_image() {
    local IMAGE="$1"
    if docker image inspect "$IMAGE" &>/dev/null; then
        log_info "镜像已存在: $IMAGE"
    else
        log_warn "拉取镜像: $IMAGE ..."
        if ! docker pull "$IMAGE"; then
            log_error "拉取失败！请设置 Docker 镜像加速:"
            log_error "  Docker Desktop → Settings → Docker Engine → registry-mirrors"
            log_error '  ["https://docker.1ms.run","https://docker.xuanyuan.me"]'
            exit 1
        fi
        log_info "拉取成功: $IMAGE"
    fi
}

ensure_docker_running() {
    if docker info &>/dev/null 2>&1; then return 0; fi
    log_warn "启动 Docker Desktop..."
    open -a Docker
    echo -n "  等待就绪"
    for i in $(seq 1 60); do
        if docker info &>/dev/null 2>&1; then echo ""; log_info "Docker Desktop 已就绪"; return 0; fi
        echo -n "."; sleep 2
    done
    echo ""; log_error "Docker Desktop 启动超时"; exit 1
}

# 检测端口是否已被 Docker 容器占用 (不管容器名)
# 返回占用该端口的容器名, 没有则返回空
get_container_on_port() {
    local PORT=$1
    docker ps --format '{{.Names}}\t{{.Ports}}' 2>/dev/null | grep "0.0.0.0:${PORT}->" | awk -F'\t' '{print $1}' | head -1
}

ensure_mongo() {
    local C="ymlink-mongo" IMG="mongo:6.0"

    # 先检查是否有任何容器已在用 27017 端口 (不管名字)
    local EXISTING=$(get_container_on_port 27017)
    if [ -n "$EXISTING" ]; then
        log_info "MongoDB 已运行 (容器: $EXISTING, 端口: 27017, 保持不变)"
        # 记录实际容器名，供后续初始化用
        MONGO_CONTAINER="$EXISTING"
        return 0
    fi

    # 没有运行中的，检查是否有同名已停止容器
    if docker ps -a --format '{{.Names}}' | grep -q "^${C}$"; then
        log_info "启动已有 MongoDB 容器 ($C)..."
        docker start "$C" 2>/dev/null
    else
        ensure_docker_image "$IMG"
        log_info "创建 MongoDB (认证: admin/admin)..."
        docker run -d --name "$C" -p 27017:27017 -v ymlink-mongo-data:/data/db \
            -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=admin \
            "$IMG" --auth
    fi
    MONGO_CONTAINER="$C"

    echo -n "  等待 MongoDB"
    for i in $(seq 1 40); do
        if docker exec "$MONGO_CONTAINER" mongosh --quiet --eval "db.runCommand({ping:1})" -u admin -p admin --authenticationDatabase admin &>/dev/null; then
            echo ""; log_info "MongoDB 就绪 (localhost:27017)"; return 0
        fi
        echo -n "."; sleep 1
    done
    echo ""; log_error "MongoDB 超时！docker logs $MONGO_CONTAINER"; exit 1
}

ensure_influx() {
    local C="ymlink-influx" IMG="influxdb:2.7"

    # 先检查是否有任何容器已在用 8086 端口
    local EXISTING=$(get_container_on_port 8086)
    if [ -n "$EXISTING" ]; then
        log_info "InfluxDB 已运行 (容器: $EXISTING, 端口: 8086, 保持不变)"
        INFLUX_CONTAINER="$EXISTING"
        return 0
    fi

    if docker ps -a --format '{{.Names}}' | grep -q "^${C}$"; then
        log_info "启动已有 InfluxDB 容器 ($C)..."
        docker start "$C" 2>/dev/null
    else
        ensure_docker_image "$IMG"
        log_info "创建 InfluxDB (org: ymlink)..."
        docker run -d --name "$C" -p 8086:8086 -v ymlink-influx-data:/var/lib/influxdb2 \
            -e DOCKER_INFLUXDB_INIT_MODE=setup \
            -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=admin12345678 \
            -e DOCKER_INFLUXDB_INIT_ORG=ymlink \
            -e DOCKER_INFLUXDB_INIT_BUCKET=realtime \
            -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=ymlink-influx-token \
            -e DOCKER_INFLUXDB_INIT_RETENTION=168h \
            "$IMG"
    fi
    INFLUX_CONTAINER="$C"

    echo -n "  等待 InfluxDB"
    for i in $(seq 1 40); do
        if curl -s http://localhost:8086/health 2>/dev/null | grep -q '"status":"pass"'; then
            echo ""; log_info "InfluxDB 就绪 (localhost:8086)"; return 0
        fi
        echo -n "."; sleep 1
    done
    echo ""; log_error "InfluxDB 超时！docker logs $INFLUX_CONTAINER"; exit 1
}

# ============================================================
# [1/8] 检查全部依赖
# ============================================================
log_step "1/8  检查依赖"

ERRORS=0

if [ ! -d "$SRC_DIR" ]; then
    log_error "未找到源码目录: $SRC_DIR"
    log_error "请确保目录结构: ymlink-q2-new-master/ 和 ymlink-q2-env/ 平级放置"
    exit 1
fi
[ ! -f "$SRC_DIR/go.mod" ] && { log_error "源码不完整，缺少 go.mod"; exit 1; }
log_info "源码目录: $SRC_DIR"

command -v go &>/dev/null && log_info "Go: $(go version | awk '{print $3}')" || { log_error "未安装 Go (brew install go)"; ERRORS=$((ERRORS+1)); }

HAS_NODE=false
if command -v node &>/dev/null && command -v npm &>/dev/null; then
    log_info "Node: $(node -v) / npm: $(npm -v)"; HAS_NODE=true
else
    log_warn "未安装 Node.js (brew install node)，前端无法启动"
fi

command -v docker &>/dev/null && log_info "Docker: $(docker --version | awk '{print $3}' | tr -d ',')" || { log_error "未安装 Docker"; ERRORS=$((ERRORS+1)); }

# Go 补丁
[ -f "$PATCH_DIR/apps/server/main.go" ] && [ -f "$PATCH_DIR/plugin/plugin.http.server.go" ] && log_info "Go 补丁文件就绪" || { log_error "缺少 Go 补丁文件 (patches/)"; ERRORS=$((ERRORS+1)); }

# 前端补丁
if [ -d "$WEB_PATCH_DIR" ]; then
    WEB_PATCH_COUNT=$(find "$WEB_PATCH_DIR" -type f ! -name '.DS_Store' | wc -l | tr -d ' ')
    log_info "前端补丁: $WEB_PATCH_COUNT 个文件"
else
    log_info "无前端补丁 (patches/.web/ 不存在)"
fi

# ip2region
[ -f "$ENV_DIR/ip2region.xdb" ] || [ -f "$SRC_DIR/data/ip2region/ip2region.xdb" ] && log_info "ip2region.xdb 就绪" || { log_error "缺少 ip2region.xdb"; ERRORS=$((ERRORS+1)); }

[ $ERRORS -gt 0 ] && { log_error "有 $ERRORS 项检查未通过，请修复后重试"; exit 1; }
log_info "全部依赖检查通过 ✓"

# ============================================================
# [2/8] 检查端口
# ============================================================
log_step "2/8  检查端口"

check_port 8080  "Go后端"
[ "$HAS_NODE" = "true" ] && [ -d "$WEB_DIR" ] && check_port 3000 "Vue前端"
# 注: 27017/8086 端口由 Docker 容器管理，不在此检查
log_info "全部端口就绪 ✓"

# ============================================================
# [3/8] 启动 Docker 服务
# ============================================================
log_step "3/8  启动 Docker 服务 (自动检测/拉取/创建)"

ensure_docker_running
ensure_mongo
ensure_influx

# ============================================================
# [4/8] 初始化数据库 + 验证
# ============================================================
log_step "4/8  初始化数据库"

MONGO_INIT_FLAG="$ENV_DIR/.mongo_initialized"
if [ ! -f "$MONGO_INIT_FLAG" ]; then
    log_info "初始化 MongoDB 集合..."
    docker exec "$MONGO_CONTAINER" mongosh -u admin -p admin --authenticationDatabase admin --quiet --eval '
        db = db.getSiblingDB("ymlink_q2");
        var cols = ["account","account_key","drive_record","event","friend","friend_group",
            "friend_label","friend_message","group_member","quest","quest_group",
            "quest_template","robot","robot_group","setting","system_log",
            "task","task_template","user","vps","worker"];
        var c=0; cols.forEach(function(n){if(!db.getCollectionNames().includes(n)){db.createCollection(n);c++}});
        print("新建 "+c+" 个集合, 总计 "+db.getCollectionNames().length+" 个");
    ' 2>/dev/null && touch "$MONGO_INIT_FLAG" || log_warn "集合初始化可能部分失败"
else
    log_info "MongoDB 已初始化"
fi

# 验证 MongoDB 连通
if docker exec "$MONGO_CONTAINER" mongosh --quiet "mongodb://admin:admin@localhost:27017/ymlink_q2?authSource=admin" --eval 'db.getCollectionNames().length' &>/dev/null; then
    log_info "MongoDB 连通验证通过 ✓"
else
    log_error "MongoDB 连通验证失败！"; exit 1
fi

INFLUX_INIT_FLAG="$ENV_DIR/.influx_initialized"
if [ ! -f "$INFLUX_INIT_FLAG" ]; then
    docker exec "$INFLUX_CONTAINER" influx bucket create --name history --org ymlink --retention 8760h --token ymlink-influx-token 2>/dev/null || true
    touch "$INFLUX_INIT_FLAG"
else
    log_info "InfluxDB 已初始化"
fi

if curl -s -H "Authorization: Token ymlink-influx-token" http://localhost:8086/api/v2/buckets?org=ymlink 2>/dev/null | grep -q '"name"'; then
    log_info "InfluxDB 连通验证通过 ✓"
else
    log_warn "InfluxDB 连通验证未通过 (可能仍在启动)"
fi

# ============================================================
# [5/8] 准备运行时目录和数据文件
# ============================================================
log_step "5/8  准备运行时目录和数据文件"

cd "$SRC_DIR"

# 全部必需目录 (根据源码分析)
REQUIRED_DIRS=(
    "data/friendb" "data/ip2region" "log"
    "file/task" "file/login" "file/material" "file/message"
    "file/usedb" "file/qzonedb" "file/materialdb" "file/realinfodb"
    "file/android_pack" "file/ios_pack" "file/ini_pack"
    "apps/server/file"
)
NEW=0
for d in "${REQUIRED_DIRS[@]}"; do
    [ ! -d "$d" ] && { mkdir -p "$d"; NEW=$((NEW+1)); }
done
log_info "运行时目录: ${#REQUIRED_DIRS[@]} 个 (新建 $NEW 个)"

# ip2region.xdb
IP2REGION_DEST="$SRC_DIR/data/ip2region/ip2region.xdb"
if [ ! -f "$IP2REGION_DEST" ]; then
    cp "$ENV_DIR/ip2region.xdb" "$IP2REGION_DEST"
    log_info "ip2region.xdb 已拷贝 ($(du -h "$IP2REGION_DEST" | awk '{print $1}'))"
else
    log_info "ip2region.xdb 已存在 ($(du -h "$IP2REGION_DEST" | awk '{print $1}'))"
fi

# friendb
FRIENDB_FILE="$SRC_DIR/data/friendb/friends.friendb"
[ ! -f "$FRIENDB_FILE" ] && touch "$FRIENDB_FILE" && log_info "friends.friendb 创建 (启动后自动扩展)"

# html/
HTML_DIR="$SRC_DIR/apps/server/html"
if [ -f "$HTML_DIR/index.html" ]; then
    log_info "前端 html/ 就绪 (有 index.html)"
else
    mkdir -p "$HTML_DIR"
    log_warn "html/ 缺少 index.html, 后端 API 可用但页面可能不可用"
fi

# ============================================================
# [6/8] 下载 Go 依赖
# ============================================================
log_step "6/8  检查/下载 Go 依赖"

cd "$SRC_DIR"
export GOPROXY=https://goproxy.cn,direct
log_info "检查 Go 模块依赖..."
go mod download 2>&1 | tail -5 || true
log_info "Go 依赖就绪 ✓"

# ============================================================
# [7/8] 启动 Go 后端 (go run -overlay)
# ============================================================
log_step "7/8  启动 Go 后端"

# 生成 overlay.json
OVERLAY_JSON="$PATCH_DIR/overlay.json"
cat > "$OVERLAY_JSON" << EOFOVERLAY
{
  "Replace": {
    "$SRC_DIR/apps/server/main.go": "$PATCH_DIR/apps/server/main.go",
    "$SRC_DIR/plugin/plugin.http.server.go": "$PATCH_DIR/plugin/plugin.http.server.go"
  }
}
EOFOVERLAY
log_info "Go overlay.json 已生成 (2 个补丁文件)"

# 环境变量
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

log_info "启动 Go 后端 (go run -overlay, 首次需编译请等待)..."
log_info "日志: log/ymlink-q2.log | log/access.log | log/error.log"

cd "$SRC_DIR"
go run -overlay="$OVERLAY_JSON" ./apps/server/ &
BACKEND_PID=$!
PIDS+=($BACKEND_PID)
log_info "Go 后端 PID: $BACKEND_PID"

# 等后端启动
BACKEND_READY=false
echo -n "  等待后端就绪"
for i in $(seq 1 90); do
    if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
        echo ""
        log_error "Go 后端进程已退出！查看日志: cat $SRC_DIR/log/ymlink-q2.log"
        break
    fi
    if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/ 2>/dev/null | grep -qE "200|404|401|302"; then
        echo ""
        BACKEND_READY=true
        log_info "Go 后端就绪 ✓ (localhost:8080)"
        break
    fi
    echo -n "."
    sleep 2
done
if [ "$BACKEND_READY" = "false" ] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo ""
    log_warn "后端仍在编译/启动中..."
fi

# ============================================================
# [8/8] 启动 Vue 前端 (含补丁)
# ============================================================
log_step "8/8  启动 Vue 前端"

if [ "$HAS_NODE" = "true" ] && [ -d "$WEB_DIR" ]; then
    # 应用前端补丁
    apply_web_patches

    cd "$WEB_DIR"
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖 (npm install)..."
        npm install
    else
        log_info "前端依赖已安装"
    fi
    log_info "启动 Vue 前端 (端口 3000, 热更新)..."
    npm run dev &
    FRONTEND_PID=$!
    PIDS+=($FRONTEND_PID)
    log_info "Vue 前端 PID: $FRONTEND_PID"
elif [ "$HAS_NODE" = "false" ]; then
    log_warn "未安装 Node.js，跳过前端"
elif [ ! -d "$WEB_DIR" ]; then
    log_warn "未找到 .web/ 目录"
    log_warn "可访问 http://localhost:8080 使用已编译前端"
fi

# ============================================================
# 启动完成
# ============================================================
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   YMLink-Q2 一键启动完成 (开发模式)             ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}服务:${NC}"
echo -e "    后端 API:        ${CYAN}http://localhost:8080${NC}"
if [ "$HAS_NODE" = "true" ] && [ -d "$WEB_DIR" ]; then
echo -e "    前端 (热更新):   ${CYAN}http://localhost:3000/?url=localhost:8080${NC}"
fi
echo -e "    InfluxDB:        ${CYAN}http://localhost:8086${NC}"
echo ""
echo -e "  ${YELLOW}⚠ 访问前端请使用:${NC}"
echo -e "    ${CYAN}http://localhost:8080/?url=localhost:8080${NC}"
echo -e "    (必须带 ?url=localhost:8080 参数，否则 API 会请求远程服务器)"
echo -e "    账号: admin  密码: a12345677"
echo ""
echo -e "  ${CYAN}补丁:${NC}"
echo -e "    Go 后端:         overlay 机制 (不修改源码)"
if [ ${#PATCHED_FILES[@]} -gt 0 ]; then
echo -e "    Vue 前端:        ${#PATCHED_FILES[@]} 个文件已补丁 (Ctrl+C 自动还原)"
fi
echo ""
echo -e "  ${CYAN}日志:${NC}"
echo -e "    tail -f $SRC_DIR/log/ymlink-q2.log"
echo -e "    tail -f $SRC_DIR/log/access.log"
echo ""
echo -e "  ${YELLOW}Ctrl+C 停止服务 + 还原前端补丁${NC}"
echo ""

wait
