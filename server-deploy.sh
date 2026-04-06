#!/bin/bash
# ============================================================
# YMLink-Q2 服务器一键部署 (重装系统后执行)
#
# 用法:
#   bash /root/env/server-deploy.sh
#
# ★ 项目结构（司法鉴定要求：原始代码不可修改）:
#   /root/q2/ymlink-q2-new-master/  ← Go 后端原始源码（不可修改）
#   /root/q2/ymlink-q2-ui-main/     ← Vue 前端原始源码（不可修改）
#   /root/q2/q2-env/                ← 补丁仓库（git）
#     ├── patches/                  ← Go overlay 补丁 + 前端补丁
#     │   ├── *.go                  ← Go 补丁文件（overlay 注入）
#     │   ├── .web/                 ← 前端补丁文件（rsync 覆盖后编译）
#     │   ├── build-patched.sh      ← Go overlay 编译脚本
#     │   └── overlay.json          ← 自动生成的 overlay 映射
#     ├── web-env/                  ← 前端编译产物（vite build 输出）
#     ├── q2-env-patch              ← Go 编译产物（overlay 编译输出）
#     └── docker-compose.yml        ← 数据库容器定义
#
# ★ 编译流程:
#   后端: go build -overlay=overlay.json → 原始源码不动，补丁注入
#   前端: rsync 补丁到 UI 源码副本 → vite build → 输出到 web-env/
#
# 部署完成后日常更新:
#   bash /root/q2/update.sh
# ============================================================
set -e

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "  ${GREEN}[OK]${NC}  $1"; }
log_warn()  { echo -e "  ${YELLOW}[!]${NC}   $1"; }
log_error() { echo -e "  ${RED}[ERR]${NC} $1"; exit 1; }
log_step()  { echo -e "\n${CYAN}[$1]${NC} $2"; }

# ============================================================
# ★ 目录配置（必须）
# ============================================================
BASE="/root/q2"

# 原始源码目录（司法鉴定用，不可修改）
SRC_GO="$BASE/ymlink-q2-new-master"       # Go 后端原始源码
SRC_UI="$BASE/ymlink-q2-ui-main"           # Vue 前端原始源码

# 补丁仓库（git 管理）
ENV="$BASE/q2-env"
PATCHES="$ENV/patches"

# 运行时数据目录
DATA="$BASE/server-data"

# 编译输出
BINARY="$ENV/q2-env-patch"                 # Go overlay 编译产物
WEB_DIST="$ENV/web-env"                    # 前端 vite build 产物

GIT_REPO="https://github.com/yunxia100/q2-env.git"

echo ""
echo -e "${GREEN}══════════════════════════════════════════${NC}"
echo -e "${GREEN}  YMLink-Q2 一键部署${NC}"
echo -e "${GREEN}  (司法鉴定版 - 原始代码不修改)${NC}"
echo -e "${GREEN}══════════════════════════════════════════${NC}"

# ============================================================
# [0] 检查源码目录
# ============================================================
log_step "0/8" "检查源码目录"

[ ! -d "$SRC_GO" ] && log_error "Go 源码目录不存在: $SRC_GO"
[ ! -f "$SRC_GO/go.mod" ] && log_error "Go 源码目录无效（缺少 go.mod）: $SRC_GO"
log_info "Go 后端源码: $SRC_GO"

[ ! -d "$SRC_UI" ] && log_error "前端源码目录不存在: $SRC_UI"
[ ! -f "$SRC_UI/package.json" ] && log_error "前端源码目录无效（缺少 package.json）: $SRC_UI"
log_info "Vue 前端源码: $SRC_UI"

# ============================================================
# [1] 安装依赖 (Docker + Git + Go + Node)
# ============================================================
log_step "1/8" "安装依赖"

export DEBIAN_FRONTEND=noninteractive

# Git + 基础工具
if command -v git &>/dev/null; then
    log_info "Git 已安装"
else
    apt-get update -qq && apt-get install -y -qq git curl jq rsync
    log_info "Git 安装完成"
fi

# rsync（补丁覆盖用）
if ! command -v rsync &>/dev/null; then
    apt-get install -y -qq rsync 2>/dev/null || true
fi

# Docker
if command -v docker &>/dev/null; then
    log_info "Docker 已安装: $(docker --version 2>/dev/null | awk '{print $3}' | tr -d ',')"
else
    echo "  安装 Docker..."
    apt-get install -y -qq docker.io 2>/dev/null || (curl -fsSL https://get.docker.com | sh)
    systemctl enable docker && systemctl start docker
    log_info "Docker 安装完成"
fi

# docker-compose
if command -v docker-compose &>/dev/null; then
    log_info "Docker Compose 就绪"
elif docker compose version &>/dev/null; then
    log_info "Docker Compose (v2 plugin) 就绪"
else
    apt-get install -y -qq docker-compose 2>/dev/null || \
    apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
    command -v docker-compose &>/dev/null || docker compose version &>/dev/null || log_error "Docker Compose 安装失败"
    log_info "Docker Compose 安装完成"
fi

# Go
if command -v go &>/dev/null; then
    log_info "Go 已安装: $(go version 2>/dev/null | awk '{print $3}')"
else
    echo "  安装 Go..."
    GO_VERSION="1.21.13"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz 2>/dev/null || \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    export PATH=$PATH:/usr/local/go/bin
    go version &>/dev/null || log_error "Go 安装失败"
    log_info "Go 安装完成: $(go version | awk '{print $3}')"
fi
export GOPATH=/root/go
export GOPROXY=https://goproxy.cn,direct

# Node.js + npm
if command -v node &>/dev/null; then
    log_info "Node.js 已安装: $(node -v)"
else
    echo "  安装 Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash - 2>/dev/null
    apt-get install -y -qq nodejs 2>/dev/null
    node -v &>/dev/null || log_error "Node.js 安装失败"
    log_info "Node.js 安装完成: $(node -v)"
fi

# ============================================================
# [2] 克隆/拉取 q2-env 补丁仓库
# ============================================================
log_step "2/8" "拉取补丁仓库"

mkdir -p "$BASE"
if [ -d "$ENV/.git" ]; then
    log_info "q2-env 仓库已存在，拉取最新..."
    cd "$ENV" && git pull
else
    log_info "克隆 q2-env 仓库..."
    rm -rf "$ENV"
    git clone "$GIT_REPO" "$ENV"
fi

[ ! -d "$PATCHES" ] && log_error "补丁目录不存在: $PATCHES"
log_info "补丁仓库就绪: $ENV"

# ============================================================
# [3] Go overlay 编译（后端）
# ============================================================
log_step "3/8" "Go overlay 编译（原始源码不修改）"

echo "  源码: $SRC_GO"
echo "  补丁: $PATCHES"
echo "  输出: $BINARY"

# 自动生成 overlay.json — 扫描补丁目录中所有 .go 文件
OVERLAY_JSON="$PATCHES/overlay.json"
FIRST=true
GO_PATCH_COUNT=0
{
    echo '{'
    echo '  "Replace": {'
    while IFS= read -r patch_file; do
        rel="${patch_file#$PATCHES/}"
        original="$SRC_GO/$rel"
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo ','
        fi
        printf '    "%s": "%s"' "$original" "$patch_file"
        GO_PATCH_COUNT=$((GO_PATCH_COUNT + 1))
    done < <(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | sort)
    echo ''
    echo '  }'
    echo '}'
} > "$OVERLAY_JSON"

log_info "overlay.json 已生成（$GO_PATCH_COUNT 个 Go 补丁文件）"

# 编译（overlay 注入，不修改原始源码）
cd "$SRC_GO"
go build -overlay="$OVERLAY_JSON" -o "$BINARY" ./apps/server/
chmod +x "$BINARY"
log_info "Go 编译成功: $BINARY ($(ls -lh "$BINARY" | awk '{print $5}'))"

# ============================================================
# [4] 前端编译（补丁覆盖 + vite build）
# ============================================================
log_step "4/8" "前端编译（补丁覆盖 → vite build）"

WEB_PATCHES="$PATCHES/.web"
if [ -d "$WEB_PATCHES" ]; then
    # 创建前端工作副本（不动原始源码）
    UI_WORK="$BASE/.ui-build-temp"
    rm -rf "$UI_WORK"
    cp -a "$SRC_UI" "$UI_WORK"
    log_info "前端工作副本: $UI_WORK"

    # rsync 补丁覆盖到工作副本
    rsync -a --exclude='*.md' "$WEB_PATCHES/" "$UI_WORK/"
    WEB_PATCH_COUNT=$(find "$WEB_PATCHES" -type f | wc -l)
    log_info "前端补丁覆盖完成（$WEB_PATCH_COUNT 个文件）"

    # 安装依赖 + 编译
    cd "$UI_WORK"
    npm install --prefer-offline 2>/dev/null || npm install
    npx vite build --outDir "$WEB_DIST"
    log_info "前端 vite build 完成: $WEB_DIST"

    # 清理工作副本
    rm -rf "$UI_WORK"
    log_info "前端工作副本已清理（原始源码未修改）"
else
    log_warn "无前端补丁目录 ($WEB_PATCHES)，使用仓库已有的 web-env/"
fi

# ============================================================
# [5] 启动数据库 (Docker)
# ============================================================
log_step "5/8" "启动数据库"

# 配置 Docker 镜像加速（国内）
if [ ! -f /etc/docker/daemon.json ] || ! grep -q "registry-mirrors" /etc/docker/daemon.json 2>/dev/null; then
    echo '{"registry-mirrors":["https://dockerpull.org","https://docker.rainbond.cc","https://docker.1ms.run"]}' > /etc/docker/daemon.json
    systemctl restart docker
    sleep 3
fi

cd "$ENV"

# 优先使用 docker-compose v1，回退到 docker compose v2
if command -v docker-compose &>/dev/null; then
    COMPOSE_CMD="docker-compose"
else
    COMPOSE_CMD="docker compose"
fi

$COMPOSE_CMD up -d mongodb influxdb

# 等 MongoDB
echo -n "  等待 MongoDB"
for i in $(seq 1 60); do
    docker exec q2-mongo mongosh --quiet --eval 'db.runCommand({ping:1})' \
        -u admin -p admin --authenticationDatabase admin >/dev/null 2>&1 && { echo ""; log_info "MongoDB 就绪"; break; }
    echo -n "."; sleep 2
    [ $i -eq 60 ] && { echo ""; log_error "MongoDB 启动超时"; }
done

# 等 InfluxDB
echo -n "  等待 InfluxDB"
for i in $(seq 1 40); do
    curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass && { echo ""; log_info "InfluxDB 就绪"; break; }
    echo -n "."; sleep 2
    [ $i -eq 40 ] && { echo ""; log_error "InfluxDB 启动超时"; }
done

# 创建 history bucket
docker exec q2-influx influx bucket create \
    --name history --org q2org --retention 8760h \
    --token q2-influx-token 2>/dev/null || true
log_info "InfluxDB history bucket 就绪"

# ============================================================
# [6] 初始化 server-data 目录
# ============================================================
log_step "6/8" "初始化运行时目录"

mkdir -p "$DATA"
cd "$DATA"

for d in data/friendb data/ip2region log html \
    file/task file/login file/material file/materialdb file/message \
    file/usedb file/qzonedb file/realinfodb file/android_pack \
    file/ios_pack file/ini_pack; do
    mkdir -p "$d"
done

# ip2region
if [ -f "$ENV/ip2region.xdb" ] && [ ! -f "$DATA/data/ip2region/ip2region.xdb" ]; then
    cp "$ENV/ip2region.xdb" "$DATA/data/ip2region/ip2region.xdb"
fi
log_info "ip2region.xdb 就绪"

# friendb
[ ! -f "$DATA/data/friendb/friends.friendb" ] && touch "$DATA/data/friendb/friends.friendb"

# 前端 (软链接到编译产物，git pull + 重新编译即更新)
rm -rf "$DATA/html/assets" "$DATA/html/index.html"
ln -sf "$WEB_DIST/assets" "$DATA/html/assets"
ln -sf "$WEB_DIST/index.html" "$DATA/html/index.html"
# images 目录也需要链接
[ -d "$WEB_DIST/images" ] && ln -sf "$WEB_DIST/images" "$DATA/html/images"
log_info "前端软链接到 $WEB_DIST"

# ============================================================
# [7] 创建 start.sh / update.sh / build.sh
# ============================================================
log_step "7/8" "创建启动/更新/编译脚本"

# --- start.sh ---
cat > "$BASE/start.sh" << 'EOF'
#!/bin/bash
cd /root/q2/server-data

export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_THREAD_LIMIT=1 SYSTEM_MSG_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=q2-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./html
export MONGO1_URL=localhost:27017/q2_db?authSource=admin MONGO1_DATABASE=q2_db
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=q2org INFLUX1_TOKEN=q2-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING=server1#client#8.9.80#http://8.130.31.166:8098
export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

pkill -f q2-env-patch 2>/dev/null || true
sleep 1
nohup /root/q2/q2-env/q2-env-patch > /root/q2/server.log 2>&1 &
echo "PID: $!"
EOF
chmod +x "$BASE/start.sh"
log_info "start.sh 已创建"

# --- build.sh (服务器端 overlay 编译 + 前端编译) ---
cat > "$BASE/build.sh" << 'BUILDEOF'
#!/bin/bash
# ============================================================
# 服务器端一键编译（Go overlay + 前端 vite build）
# 原始源码不修改，补丁通过 overlay 注入
# ============================================================
set -e

BASE="/root/q2"
SRC_GO="$BASE/ymlink-q2-new-master"
SRC_UI="$BASE/ymlink-q2-ui-main"
ENV="$BASE/q2-env"
PATCHES="$ENV/patches"
BINARY="$ENV/q2-env-patch"
WEB_DIST="$ENV/web-env"

export PATH=$PATH:/usr/local/go/bin
export GOPATH=/root/go
export GOPROXY=https://goproxy.cn,direct

echo "=========================================="
echo "  YMLink-Q2 服务器端编译"
echo "  Go 源码:  $SRC_GO（不修改）"
echo "  前端源码: $SRC_UI（不修改）"
echo "  补丁目录: $PATCHES"
echo "=========================================="

# --- Go overlay 编译 ---
echo ""
echo "[1/2] Go overlay 编译..."
OVERLAY_JSON="$PATCHES/overlay.json"
FIRST=true; COUNT=0
{
    echo '{'
    echo '  "Replace": {'
    while IFS= read -r patch_file; do
        rel="${patch_file#$PATCHES/}"
        original="$SRC_GO/$rel"
        [ "$FIRST" = true ] && FIRST=false || echo ','
        printf '    "%s": "%s"' "$original" "$patch_file"
        COUNT=$((COUNT + 1))
    done < <(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | sort)
    echo ''
    echo '  }'
    echo '}'
} > "$OVERLAY_JSON"
echo "  Go 补丁: $COUNT 个文件"

cd "$SRC_GO"
go build -overlay="$OVERLAY_JSON" -o "$BINARY" ./apps/server/
chmod +x "$BINARY"
echo "  编译成功: $BINARY ($(ls -lh "$BINARY" | awk '{print $5}'))"

# --- 前端编译 ---
echo ""
echo "[2/2] 前端编译..."
WEB_PATCHES="$PATCHES/.web"
if [ -d "$WEB_PATCHES" ]; then
    UI_WORK="$BASE/.ui-build-temp"
    rm -rf "$UI_WORK"
    cp -a "$SRC_UI" "$UI_WORK"
    rsync -a --exclude='*.md' "$WEB_PATCHES/" "$UI_WORK/"
    cd "$UI_WORK"
    npm install --prefer-offline 2>/dev/null || npm install
    npx vite build --outDir "$WEB_DIST"
    rm -rf "$UI_WORK"
    echo "  前端编译成功: $WEB_DIST"
else
    echo "  无前端补丁，跳过"
fi

echo ""
echo "=========================================="
echo "  编译完成！原始源码未做任何修改。"
echo "  运行: bash /root/q2/start.sh"
echo "=========================================="
BUILDEOF
chmod +x "$BASE/build.sh"
log_info "build.sh 已创建（服务器端编译脚本）"

# --- update.sh ---
cat > "$BASE/update.sh" << 'EOF'
#!/bin/bash
# 一键拉取补丁 + 重新编译 + 重启
set -e
echo "=== 拉取最新补丁 ==="
cd /root/q2/q2-env && git pull

echo "=== 重新编译（overlay 注入）==="
bash /root/q2/build.sh

echo "=== 重启服务 ==="
bash /root/q2/start.sh

sleep 2
if pgrep -f q2-env-patch > /dev/null; then
    echo "=== 更新完成，服务运行中 ==="
else
    echo "=== 启动失败，查看日志: tail -f /root/q2/server.log ==="
fi
EOF
chmod +x "$BASE/update.sh"
log_info "update.sh 已创建（拉取 + 编译 + 重启）"

# ============================================================
# [8] 启动服务
# ============================================================
log_step "8/8" "启动服务"

bash "$BASE/start.sh"
sleep 3

if pgrep -f q2-env-patch > /dev/null; then
    PID=$(pgrep -f q2-env-patch)
    PUBLIC_IP=$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    echo ""
    echo -e "${GREEN}══════════════════════════════════════════${NC}"
    echo -e "${GREEN}  部署完成!${NC}"
    echo -e "${GREEN}══════════════════════════════════════════${NC}"
    echo ""
    echo -e "  地址:   ${CYAN}http://${PUBLIC_IP}:8080${NC}"
    echo -e "  账号:   admin"
    echo -e "  密码:   a12345677"
    echo -e "  PID:    $PID"
    echo ""
    echo -e "  ${CYAN}目录结构:${NC}"
    echo -e "    $SRC_GO/  ← Go 原始源码（不可修改）"
    echo -e "    $SRC_UI/  ← Vue 原始源码（不可修改）"
    echo -e "    $ENV/patches/  ← 补丁文件（overlay 注入）"
    echo ""
    echo -e "  ${YELLOW}常用命令:${NC}"
    echo -e "    bash /root/q2/start.sh    # 启动服务"
    echo -e "    bash /root/q2/build.sh    # 重新编译（overlay + vite）"
    echo -e "    bash /root/q2/update.sh   # 拉取补丁 + 编译 + 重启"
    echo -e "    tail -f /root/q2/server.log  # 查看日志"
    echo -e "    pkill -f q2-env-patch     # 停止服务"
    echo ""
else
    log_warn "服务可能未启动成功"
    echo "  查看日志: tail -f $BASE/server.log"
fi

# 防火墙
if command -v ufw &>/dev/null && ufw status 2>/dev/null | grep -q "active"; then
    ufw allow 8080/tcp 2>/dev/null || true
fi
