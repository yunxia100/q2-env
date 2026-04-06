#!/bin/bash
# ============================================================
# YMLink-Q2 服务器一键启动脚本 (全新服务器可用)
#
# 自动完成:
#   ① Docker 安装 (国内镜像)
#   ② MongoDB 4.4 + InfluxDB 2.7 部署 (Docker)
#   ③ Go 安装 (国内镜像 golang.google.cn)
#   ④ Node.js 安装 (国内镜像 npmmirror.com)
#   ⑤ 底层驱动连通性检查
#   ⑥ 运行时目录 + FriendB (mmap) 初始化
#   ⑦ 前端检查
#   ⑧ Go 后端编译 (overlay 补丁)
#   ⑨ 启动服务 + 全面健康检查
#
# 目录结构:
#   /root/q2/server-start.sh            ← 本脚本
#   /root/q2/ymlink-q2-new-master/      ← 业务后端 (只读!)
#   /root/q2/ymlink-q2-ui-main/         ← 业务前端 (只读!)
#   /root/env/ymlink-q2-env/            ← 补丁和配置
#
# 用法: chmod +x /root/q2/server-start.sh && /root/q2/server-start.sh
# ============================================================

set -e

# ============ 路径配置 ============
BASE_DIR="/root/q2"
SRC_DIR="$BASE_DIR/ymlink-q2-new-master"
WEB_SRC_DIR="$BASE_DIR/ymlink-q2-ui-main"
ENV_DIR="/root/env/ymlink-q2-env"
PATCH_DIR="$ENV_DIR/patches"

# ============ 数据库配置 ============
MONGO_PORT=16010
MONGO_USER=admin
MONGO_PASS=admin_8981409
MONGO_CONTAINER=ymlink-mongo

INFLUX_PORT=16020
INFLUX_ORG=ymlink-q2
INFLUX_BUCKET=ymlink-q2
INFLUX_USER=admin
INFLUX_PASS=admin_8981409
INFLUX_TOKEN='cJ9vTXpbF0f2GhHsNQkbW9RwDPZ8qR7Y2UeiXzT4AzQ='
INFLUX_CONTAINER=ymlink-influx

# ============ 底层驱动配置 ============
# 格式: 硬件名#软件名#版本号#WebSocket地址  (多个用逗号分隔)
export DRIVE_MAPPING='server1#iOSQQ#9.1.75#http://8.130.31.166:8098'
export DRIVE_TIMEOUT=30
export DRIVE_MAX_CONN=50

# ============ 颜色 ============
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "  ${GREEN}✓${NC} $1"; }
log_warn()  { echo -e "  ${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "  ${RED}✗${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━ $1 ━━━${NC}"; }

TOTAL_STEPS=9

# ============================================================
# [1/9] 检查目录
# ============================================================
log_step "1/$TOTAL_STEPS 检查目录"

[ ! -d "$SRC_DIR" ]     && { log_error "后端目录不存在: $SRC_DIR"; exit 1; }
[ ! -f "$SRC_DIR/go.mod" ] && { log_error "后端源码不完整, 缺少 go.mod"; exit 1; }
log_info "后端: $SRC_DIR"

[ -d "$WEB_SRC_DIR" ] && log_info "前端: $WEB_SRC_DIR" || log_warn "前端目录不存在: $WEB_SRC_DIR"

[ -d "$ENV_DIR" ] && log_info "补丁: $ENV_DIR" || log_warn "补丁目录不存在: $ENV_DIR"

# ============================================================
# [2/9] 安装 Docker (国内镜像)
# ============================================================
log_step "2/$TOTAL_STEPS 检查/安装 Docker"

if command -v docker &>/dev/null; then
    log_info "Docker 已安装: $(docker --version)"
else
    log_info "安装 Docker (阿里云镜像)..."

    # 安装依赖
    apt-get update -qq >/dev/null 2>&1
    apt-get install -y -qq ca-certificates curl gnupg lsb-release >/dev/null 2>&1

    # 添加阿里云 Docker GPG key
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>/dev/null
    chmod a+r /etc/apt/keyrings/docker.gpg

    # 添加阿里云 Docker 源
    UBUNTU_CODENAME=$(. /etc/os-release && echo "$VERSION_CODENAME" 2>/dev/null || echo "jammy")
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $UBUNTU_CODENAME stable" \
        > /etc/apt/sources.list.d/docker.list

    apt-get update -qq >/dev/null 2>&1
    apt-get install -y -qq docker-ce docker-ce-cli containerd.io docker-compose-plugin >/dev/null 2>&1

    # 配置 Docker 国内镜像加速
    mkdir -p /etc/docker
    cat > /etc/docker/daemon.json << 'EOFDOCKER'
{
  "registry-mirrors": [
    "https://docker.1ms.run",
    "https://docker.xuanyuan.me"
  ]
}
EOFDOCKER

    systemctl daemon-reload
    systemctl enable docker >/dev/null 2>&1
    systemctl restart docker

    log_info "Docker $(docker --version) 安装完成"
fi

# 确保 Docker 正在运行
if ! systemctl is-active --quiet docker 2>/dev/null; then
    systemctl start docker
    log_info "Docker 服务已启动"
fi

# ============================================================
# [3/9] 部署数据库 (MongoDB + InfluxDB)
# ============================================================
log_step "3/$TOTAL_STEPS 部署数据库"

# 通用函数: 检查端口是否已被某个容器占用
get_container_on_port() {
    docker ps --format '{{.Names}} {{.Ports}}' 2>/dev/null | grep ":${1}->" | awk '{print $1}' | head -1
}

# --- MongoDB ---
MONGO_RUNNING=$(get_container_on_port $MONGO_PORT)
if [ -n "$MONGO_RUNNING" ]; then
    log_info "MongoDB 已在运行 (容器: $MONGO_RUNNING, 端口: $MONGO_PORT)"
    MONGO_CONTAINER="$MONGO_RUNNING"
elif docker ps -a --format '{{.Names}}' | grep -q "^${MONGO_CONTAINER}$"; then
    docker start "$MONGO_CONTAINER" >/dev/null
    log_info "MongoDB 已重新启动 (容器: $MONGO_CONTAINER)"
else
    log_info "部署 MongoDB 4.4 (端口: $MONGO_PORT)..."

    # 选择可用镜像
    MONGO_IMAGE=""
    if docker image inspect mongo:4.4 >/dev/null 2>&1; then
        MONGO_IMAGE="mongo:4.4"
    else
        log_info "  拉取 mongo:4.4 镜像..."
        if timeout 120 docker pull mongo:4.4 2>&1 | tail -1; then
            MONGO_IMAGE="mongo:4.4"
        fi
    fi

    # 4.4 拉不下来就用已有的 mongo 镜像
    if [ -z "$MONGO_IMAGE" ]; then
        EXISTING_MONGO=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep '^mongo:' | head -1)
        if [ -n "$EXISTING_MONGO" ]; then
            log_warn "  mongo:4.4 不可用, 使用: $EXISTING_MONGO"
            MONGO_IMAGE="$EXISTING_MONGO"
        else
            log_error "  MongoDB 镜像不可用! 请手动: docker pull mongo:4.4"
            exit 1
        fi
    fi

    docker run -d \
        --name "$MONGO_CONTAINER" \
        --restart always \
        -p ${MONGO_PORT}:27017 \
        -e MONGO_INITDB_ROOT_USERNAME="$MONGO_USER" \
        -e MONGO_INITDB_ROOT_PASSWORD="$MONGO_PASS" \
        -v ymlink-mongo-data:/data/db \
        "${MONGO_IMAGE}" >/dev/null

    log_info "MongoDB 部署完成 → localhost:$MONGO_PORT (用户: $MONGO_USER)"
fi

# --- InfluxDB ---
INFLUX_RUNNING=$(get_container_on_port $INFLUX_PORT)
if [ -n "$INFLUX_RUNNING" ]; then
    log_info "InfluxDB 已在运行 (容器: $INFLUX_RUNNING, 端口: $INFLUX_PORT)"
    INFLUX_CONTAINER="$INFLUX_RUNNING"
elif docker ps -a --format '{{.Names}}' | grep -q "^${INFLUX_CONTAINER}$"; then
    docker start "$INFLUX_CONTAINER" >/dev/null
    log_info "InfluxDB 已重新启动 (容器: $INFLUX_CONTAINER)"
else
    log_info "部署 InfluxDB 2.7 (端口: $INFLUX_PORT)..."

    if ! docker image inspect influxdb:2.7 >/dev/null 2>&1; then
        log_info "  拉取 influxdb:2.7 镜像..."
        timeout 120 docker pull influxdb:2.7 2>&1 | tail -1 || true
    fi

    if ! docker image inspect influxdb:2.7 >/dev/null 2>&1; then
        log_error "  InfluxDB 镜像不可用! 请手动: docker pull influxdb:2.7"
        exit 1
    fi

    docker run -d \
        --name "$INFLUX_CONTAINER" \
        --restart always \
        -p ${INFLUX_PORT}:8086 \
        -e DOCKER_INFLUXDB_INIT_MODE=setup \
        -e DOCKER_INFLUXDB_INIT_USERNAME="$INFLUX_USER" \
        -e DOCKER_INFLUXDB_INIT_PASSWORD="$INFLUX_PASS" \
        -e DOCKER_INFLUXDB_INIT_ORG="$INFLUX_ORG" \
        -e DOCKER_INFLUXDB_INIT_BUCKET="$INFLUX_BUCKET" \
        -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN="$INFLUX_TOKEN" \
        -v ymlink-influx-data:/var/lib/influxdb2 \
        influxdb:2.7 >/dev/null

    log_info "InfluxDB 部署完成 → localhost:$INFLUX_PORT (org: $INFLUX_ORG)"
fi

# 等待数据库就绪
log_info "等待数据库就绪..."
for i in $(seq 1 10); do
    MONGO_OK=false
    INFLUX_OK=false
    timeout 2 bash -c "echo >/dev/tcp/localhost/$MONGO_PORT" 2>/dev/null && MONGO_OK=true
    timeout 2 bash -c "echo >/dev/tcp/localhost/$INFLUX_PORT" 2>/dev/null && INFLUX_OK=true
    if [ "$MONGO_OK" = true ] && [ "$INFLUX_OK" = true ]; then
        break
    fi
    sleep 1
done

if [ "$MONGO_OK" = true ]; then
    log_info "MongoDB 就绪 (localhost:$MONGO_PORT)"
else
    log_error "MongoDB 未就绪! 请检查: docker logs $MONGO_CONTAINER"
fi

if [ "$INFLUX_OK" = true ]; then
    log_info "InfluxDB 就绪 (localhost:$INFLUX_PORT)"
else
    log_error "InfluxDB 未就绪! 请检查: docker logs $INFLUX_CONTAINER"
fi

# ============================================================
# [4/9] 检查底层驱动连通性
# ============================================================
log_step "4/$TOTAL_STEPS 检查底层驱动"

DRIVE_OK=0
DRIVE_FAIL=0
DRIVE_DETAIL=""

if [ -z "$DRIVE_MAPPING" ]; then
    log_error "DRIVE_MAPPING 未配置! 底层驱动是项目必须依赖"
    log_error "请在脚本顶部设置 DRIVE_MAPPING"
    log_error "格式: 硬件名#软件名#版本号#WebSocket地址"
    exit 1
fi

IFS=',' read -ra DRIVE_LIST <<< "$DRIVE_MAPPING"
for drive_item in "${DRIVE_LIST[@]}"; do
    IFS='#' read -ra PARTS <<< "$drive_item"
    if [ ${#PARTS[@]} -lt 4 ]; then
        log_error "驱动配置格式错误: $drive_item"
        log_error "正确格式: 硬件名#软件名#版本号#WebSocket地址"
        DRIVE_FAIL=$((DRIVE_FAIL + 1))
        continue
    fi

    D_HW="${PARTS[0]}"
    D_SW="${PARTS[1]}"
    D_VER="${PARTS[2]}"
    D_URL="${PARTS[3]}"

    # 从 URL 提取 host:port
    D_HOST=$(echo "$D_URL" | sed -E 's|https?://||' | sed 's|/.*||')
    D_IP=$(echo "$D_HOST" | cut -d: -f1)
    D_PORT=$(echo "$D_HOST" | cut -d: -f2)

    log_info "驱动: ${D_HW} / ${D_SW} v${D_VER} → ${D_URL}"

    # TCP 连通性测试
    if timeout 5 bash -c "echo >/dev/tcp/$D_IP/$D_PORT" 2>/dev/null; then
        log_info "  连通 ✓ (${D_IP}:${D_PORT})"
        DRIVE_OK=$((DRIVE_OK + 1))
    else
        log_error "  不通 ✗ (${D_IP}:${D_PORT})"
        log_error "  请检查驱动进程: ssh root@${D_IP} 'pgrep -la qq_mini'"
        DRIVE_FAIL=$((DRIVE_FAIL + 1))
    fi

    DRIVE_DETAIL="${DRIVE_DETAIL}\n    ${D_SW} v${D_VER} → ${D_URL}"
done

if [ $DRIVE_FAIL -gt 0 ] && [ $DRIVE_OK -eq 0 ]; then
    log_error "所有底层驱动都不可用! 项目无法正常工作"
    echo -e "  ${YELLOW}输入 y 强制继续, 其他键退出:${NC} \c"
    read -t 15 -n 1 REPLY || REPLY="n"
    echo
    if [ "$REPLY" != "y" ] && [ "$REPLY" != "Y" ]; then
        exit 1
    fi
    log_warn "强制继续 (驱动不可用, 部分功能异常)"
elif [ $DRIVE_FAIL -gt 0 ]; then
    log_warn "部分驱动不可用 ($DRIVE_OK 可用, $DRIVE_FAIL 不可用)"
else
    log_info "所有驱动连通 ($DRIVE_OK 个)"
fi

# ============================================================
# [5/9] 安装 Go (国内镜像)
# ============================================================
log_step "5/$TOTAL_STEPS 检查/安装 Go"

export PATH=/usr/local/go/bin:/opt/gopath/bin:$PATH
export GOROOT=/usr/local/go
export GOPATH=/opt/gopath
export GOPROXY=https://goproxy.cn,direct

if command -v go &>/dev/null; then
    log_info "Go 已安装: $(go version)"
else
    log_info "安装 Go 1.22.5 (国内镜像 golang.google.cn)..."
    ARCH=$(uname -m)
    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        GO_ARCH="arm64"
    else
        GO_ARCH="amd64"
    fi
    wget -q "https://golang.google.cn/dl/go1.22.5.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz
    if [ ! -s /tmp/go.tar.gz ]; then
        log_error "Go 下载失败! 请检查网络"; exit 1
    fi
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    log_info "Go $(go version) 安装完成"
fi

# ============================================================
# [6/9] 安装 Node.js (国内镜像)
# ============================================================
log_step "6/$TOTAL_STEPS 检查/安装 Node.js"

export PATH=/usr/local/node/bin:$PATH

if command -v node &>/dev/null; then
    log_info "Node.js 已安装: $(node -v)"
else
    if [ -d "$WEB_SRC_DIR" ]; then
        log_info "安装 Node.js 18 (国内镜像 npmmirror.com)..."
        ARCH=$(uname -m)
        if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
            NODE_ARCH="arm64"
        else
            NODE_ARCH="x64"
        fi
        NODE_VER="v18.20.8"
        wget -q "https://npmmirror.com/mirrors/node/${NODE_VER}/node-${NODE_VER}-linux-${NODE_ARCH}.tar.xz" -O /tmp/node.tar.xz
        if [ ! -s /tmp/node.tar.xz ]; then
            log_error "Node.js 下载失败! 请检查网络"; exit 1
        fi
        rm -rf /usr/local/node
        mkdir -p /usr/local/node
        tar -xJf /tmp/node.tar.xz -C /usr/local/node --strip-components=1
        rm -f /tmp/node.tar.xz
        export PATH=/usr/local/node/bin:$PATH
        log_info "Node.js $(node -v) 安装完成"
    else
        log_warn "无前端目录, 跳过 Node.js 安装"
    fi
fi

# npm 国内镜像
if command -v npm &>/dev/null; then
    npm config set registry https://registry.npmmirror.com 2>/dev/null || true
fi

# ============================================================
# [7/9] 准备运行时目录 + FriendB (mmap)
# ============================================================
log_step "7/$TOTAL_STEPS 准备运行时目录 + FriendB"

cd "$SRC_DIR"
for d in "data/friendb" "data/ip2region" "log" \
         "file/task" "file/login" "file/material" "file/message" \
         "file/usedb" "file/qzonedb" "file/materialdb" "file/realinfodb" \
         "file/android_pack" "file/ios_pack" "file/ini_pack" \
         "apps/server/file"; do
    mkdir -p "$d"
done

# ip2region.xdb
if [ -f "$ENV_DIR/ip2region.xdb" ] && [ ! -f "$SRC_DIR/data/ip2region/ip2region.xdb" ]; then
    cp "$ENV_DIR/ip2region.xdb" "$SRC_DIR/data/ip2region/ip2region.xdb"
    log_info "ip2region.xdb 已拷贝"
else
    log_info "ip2region.xdb 就绪"
fi

# FriendB (mmap 文件) - 创建固定大小的稀疏文件用于 mmap 映射
FRIENDB_FILE="$SRC_DIR/data/friendb/friends.friendb"
FRIENDB_TOTAL=100000
if [ ! -f "$FRIENDB_FILE" ]; then
    # 创建稀疏文件, FriendB 会通过 syscall.Mmap 映射此文件
    truncate -s 0 "$FRIENDB_FILE"
    log_info "FriendB 数据文件已创建 (mmap): $FRIENDB_FILE"
else
    FRIENDB_SIZE=$(stat -c%s "$FRIENDB_FILE" 2>/dev/null || echo "0")
    log_info "FriendB 数据文件已存在 (大小: ${FRIENDB_SIZE} bytes)"
fi

log_info "运行时目录就绪"

# ============================================================
# [8/9] 检查前端 + 编译后端
# ============================================================
log_step "8/$TOTAL_STEPS 检查前端 + 编译后端"

# --- 前端 ---
HTML_DIR="$SRC_DIR/apps/server/html"

# 优先使用源码自带的预编译前端 (已包含正确的 /api 前缀)
# ymlink-q2-ui-main 的 api.user.ts 存在 BASE_URL 路径问题, 重新编译会导致登录失败
# 如需强制重新编译: 手动删除 html/index.html 后再运行
if [ -f "$HTML_DIR/index.html" ]; then
    log_info "前端: 使用已有预编译 (html/index.html)"
elif [ -d "$WEB_SRC_DIR" ] && command -v node &>/dev/null; then
    cd "$WEB_SRC_DIR"
    [ ! -d "node_modules" ] && { log_info "安装前端依赖..."; npm install --legacy-peer-deps 2>&1 | tail -1; }
    log_info "构建前端..."
    npm run build 2>&1 | tail -3
    if [ -d "$WEB_SRC_DIR/dist" ]; then
        rm -rf "$HTML_DIR"
        cp -r "$WEB_SRC_DIR/dist" "$HTML_DIR"
        log_info "前端构建完成 → $HTML_DIR"
    else
        log_error "前端构建产物未找到"
    fi
else
    log_error "前端不可用! 无 html/index.html 且无法构建"; exit 1
fi

# --- 后端编译 ---
cd "$SRC_DIR"

OVERLAY_JSON="$PATCH_DIR/overlay.json"
if [ -d "$PATCH_DIR" ]; then
    REPLACE_ITEMS=""
    while IFS= read -r patch_file; do
        rel_path="${patch_file#$PATCH_DIR/}"
        src_file="$SRC_DIR/$rel_path"
        if [ -f "$src_file" ]; then
            [ -n "$REPLACE_ITEMS" ] && REPLACE_ITEMS="$REPLACE_ITEMS,"
            REPLACE_ITEMS="$REPLACE_ITEMS
    \"$src_file\": \"$patch_file\""
        fi
    done < <(find "$PATCH_DIR" -name "*.go" -type f)

    if [ -n "$REPLACE_ITEMS" ]; then
        cat > "$OVERLAY_JSON" << EOFOVERLAY
{
  "Replace": {$REPLACE_ITEMS
  }
}
EOFOVERLAY
        log_info "overlay.json 已生成 ($(grep -c ':' "$OVERLAY_JSON") 个补丁)"
    else
        log_warn "未找到 .go 补丁文件"
        OVERLAY_JSON=""
    fi
else
    log_warn "补丁目录不存在, 编译原始代码"
    OVERLAY_JSON=""
fi

log_info "检查 Go 依赖..."
go mod download 2>&1 | tail -3 || true

OUTPUT_BIN="$BASE_DIR/ymlink-server"
if [ -n "$OVERLAY_JSON" ] && [ -f "$OVERLAY_JSON" ]; then
    log_info "编译中 (go build -overlay, 带补丁)..."
    go build -overlay="$OVERLAY_JSON" -o "$OUTPUT_BIN" ./apps/server/
else
    log_info "编译中 (go build)..."
    go build -o "$OUTPUT_BIN" ./apps/server/
fi

if [ -f "$OUTPUT_BIN" ]; then
    log_info "编译成功 → $OUTPUT_BIN ($(du -h "$OUTPUT_BIN" | awk '{print $1}'))"
else
    log_error "编译失败!"; exit 1
fi

# ============================================================
# [9/9] 启动服务 + 健康检查
# ============================================================
log_step "9/$TOTAL_STEPS 启动服务"

# 停止旧进程
OLD_PID=$(pgrep -f "ymlink-server" 2>/dev/null || true)
if [ -n "$OLD_PID" ]; then
    kill $OLD_PID 2>/dev/null || true
    sleep 1
    log_info "已停止旧进程 (PID: $OLD_PID)"
fi

cd "$SRC_DIR"

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

export MONGO1_URL="localhost:${MONGO_PORT}/ymlink_q2?authSource=admin"
export MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME="$MONGO_USER"
export MONGO1_PASSWORD="$MONGO_PASS"

export INFLUX1_URL="http://localhost:${INFLUX_PORT}"
export INFLUX1_ORG="$INFLUX_ORG"
export INFLUX1_TOKEN="$INFLUX_TOKEN"

export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
export FRIENDB1_TOTAL=$FRIENDB_TOTAL
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb

# DRIVE_MAPPING 已在脚本顶部 export
export MOBILE_PORT=16100

# 后台启动
nohup "$OUTPUT_BIN" >> "$SRC_DIR/log/ymlink-q2.log" 2>&1 &
SERVER_PID=$!

# ============ 健康检查 ============
log_info "等待后端启动..."
sleep 3

if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    log_error "后端进程崩溃! 最后 20 行日志:"
    tail -20 "$SRC_DIR/log/ymlink-q2.log" 2>/dev/null
    exit 1
fi

HEALTH_OK=true

# 等待 HTTP 可用
for i in 1 2 3 4 5; do
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 3 http://localhost:8080/ 2>/dev/null || echo "000")
    [ "$HTTP_CODE" != "000" ] && break
    sleep 2
done

if [ "$HTTP_CODE" = "000" ]; then
    log_error "后端 HTTP 端口 8080 无响应"
    HEALTH_OK=false
else
    log_info "后端 HTTP 正常 (状态码: $HTTP_CODE)"
fi

# 检查前端页面
if curl -s http://localhost:8080/ 2>/dev/null | grep -q "YMLink"; then
    log_info "前端页面正常"
else
    log_warn "前端页面异常"
    HEALTH_OK=false
fi

# 检查登录接口
LOGIN_RESP=$(curl -s -X POST http://localhost:8080/api/user/signin \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"a12345677"}' \
    --connect-timeout 5 --max-time 10 2>/dev/null || echo "")

if echo "$LOGIN_RESP" | grep -q '"code":200'; then
    log_info "登录接口正常 (admin 登录成功)"
elif echo "$LOGIN_RESP" | grep -q '"code"'; then
    LOGIN_CODE=$(echo "$LOGIN_RESP" | grep -oP '"code":\K[0-9]+' || echo "?")
    log_warn "登录有响应但失败 (code: $LOGIN_CODE)"
else
    log_error "登录接口无响应"
    HEALTH_OK=false
fi

# ============ 最终输出 ============
PUBLIC_IP=$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")

echo ""
if [ "$HEALTH_OK" = true ]; then
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║   YMLink-Q2 启动成功! 所有检查通过                  ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
else
    echo -e "${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║   YMLink-Q2 已启动, 但部分检查未通过                ║${NC}"
    echo -e "${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"
fi

echo ""
echo -e "  ${CYAN}访问地址:${NC}   http://${PUBLIC_IP}:8080"
echo -e "  ${CYAN}登录账号:${NC}   admin"
echo -e "  ${CYAN}登录密码:${NC}   a12345677"
echo -e "  ${CYAN}进程 PID:${NC}   $SERVER_PID"
echo ""
echo -e "  ${CYAN}底层驱动:${NC}$DRIVE_DETAIL"
echo ""
echo -e "  ${CYAN}数据库:${NC}"
echo -e "    MongoDB   → localhost:$MONGO_PORT (容器: $MONGO_CONTAINER)"
echo -e "    InfluxDB  → localhost:$INFLUX_PORT (容器: $INFLUX_CONTAINER)"
echo ""
echo -e "  ${CYAN}FriendB:${NC}    $FRIENDB_FILE"
echo ""
echo -e "  ${CYAN}日志:${NC}       tail -f $SRC_DIR/log/ymlink-q2.log"
echo -e "  ${CYAN}停止服务:${NC}   kill $SERVER_PID"
echo -e "  ${CYAN}重启:${NC}       $0"
echo ""
