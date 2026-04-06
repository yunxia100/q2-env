#!/bin/bash
# ============================================================
# YMLink-Q2 Ubuntu 22.04 一键部署脚本
#
# 使用方法:
#   1. 把以下目录上传到服务器的 /opt/ymlink-q2/:
#      - ymlink-q2-new-master/   (Go 源码)
#      - ymlink-q2-master/.web/  (Vue 前端源码)
#      - ymlink-q2-env/          (补丁和配置)
#
#   2. 在服务器执行:
#      chmod +x /opt/ymlink-q2/ymlink-q2-env/deploy.sh
#      sudo /opt/ymlink-q2/ymlink-q2-env/deploy.sh
#
#   3. 之后日常启动/重启:
#      sudo systemctl restart ymlink-q2
#      sudo systemctl status ymlink-q2
#
# 默认端口: 8080
# 登录账号: admin / a12345677
# ============================================================

set -e

# ============ 颜色 ============
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "  ${GREEN}✓${NC} $1"; }
log_warn()  { echo -e "  ${YELLOW}⚠${NC} $1"; }
log_error() { echo -e "  ${RED}✗${NC} $1"; exit 1; }

# ============ 检查 root ============
if [ "$EUID" -ne 0 ]; then
    log_error "请用 sudo 执行此脚本"
fi

# ============ 路径 (可自定义) ============
BASE_DIR="/opt/ymlink-q2"
SRC_DIR="$BASE_DIR/ymlink-q2-new-master"
WEB_DIR="$BASE_DIR/ymlink-q2-master/.web"
ENV_DIR="$BASE_DIR/ymlink-q2-env"
PATCH_DIR="$ENV_DIR/patches"

# ============ 检查源码目录 ============
if [ ! -d "$SRC_DIR" ]; then
    log_error "源码目录不存在: $SRC_DIR\n  请先上传 ymlink-q2-new-master/ 到 $BASE_DIR/"
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   YMLink-Q2 服务器部署 (Ubuntu 22.04)            ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""

# ============================================================
# 第一步: 安装系统依赖
# ============================================================
echo -e "${CYAN}[1/6] 安装系统依赖...${NC}"

apt-get update -qq

# Go 1.22+
if ! command -v go &>/dev/null || [[ "$(go version 2>/dev/null)" != *"go1.2"[2-9]* && "$(go version 2>/dev/null)" != *"go1."[3-9]* ]]; then
    log_info "安装 Go 1.22..."
    GO_VERSION="1.22.5"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    # 全局环境变量
    cat > /etc/profile.d/go.sh << 'EOF'
export GOROOT=/usr/local/go
export GOPATH=/opt/gopath
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH
EOF
    source /etc/profile.d/go.sh
    log_info "Go $(go version) 安装完成"
else
    source /etc/profile.d/go.sh 2>/dev/null || true
    export PATH=/usr/local/go/bin:$PATH
    log_info "Go 已安装: $(go version)"
fi

export GOROOT=/usr/local/go
export GOPATH=/opt/gopath
export PATH=$GOROOT/bin:$GOPATH/bin:$PATH

# Node.js 18+ (用于构建前端)
if ! command -v node &>/dev/null; then
    log_info "安装 Node.js 18..."
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
    apt-get install -y -qq nodejs
    log_info "Node.js $(node -v) 安装完成"
else
    log_info "Node.js 已安装: $(node -v)"
fi

# MongoDB 7
if ! command -v mongod &>/dev/null && ! systemctl is-active --quiet mongod 2>/dev/null; then
    log_info "安装 MongoDB 7..."
    apt-get install -y -qq gnupg curl
    curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg 2>/dev/null
    echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" > /etc/apt/sources.list.d/mongodb-org-7.0.list
    apt-get update -qq
    apt-get install -y -qq mongodb-org
    systemctl enable mongod
    systemctl start mongod
    sleep 2

    # 创建 admin 用户
    mongosh admin --quiet --eval '
        db.createUser({
            user: "admin",
            pwd: "admin",
            roles: [{ role: "root", db: "admin" }]
        })
    ' 2>/dev/null || true
    log_info "MongoDB 安装完成"
else
    systemctl start mongod 2>/dev/null || true
    log_info "MongoDB 已运行"
fi

# InfluxDB 2
if ! command -v influxd &>/dev/null && ! systemctl is-active --quiet influxdb 2>/dev/null; then
    log_info "安装 InfluxDB 2..."
    wget -q https://dl.influxdata.com/influxdb/releases/influxdb2-2.7.1-amd64.deb -O /tmp/influxdb2.deb
    dpkg -i /tmp/influxdb2.deb 2>/dev/null || apt-get install -f -y -qq
    rm /tmp/influxdb2.deb
    systemctl enable influxdb
    systemctl start influxdb
    sleep 2

    # 初始化 (如果还没配置)
    influx setup --username admin --password adminadmin --org ymlink --bucket ymlink --token ymlink-influx-token --force 2>/dev/null || true
    log_info "InfluxDB 安装完成"
else
    systemctl start influxdb 2>/dev/null || true
    log_info "InfluxDB 已运行"
fi

# ============================================================
# 第二步: 准备运行时目录
# ============================================================
echo -e "${CYAN}[2/6] 准备运行时目录...${NC}"

cd "$SRC_DIR"
for d in "data/friendb" "data/ip2region" "log" "file/task" "file/login" "file/material" \
         "file/message" "file/usedb" "file/qzonedb" "file/materialdb" "file/realinfodb" \
         "apps/server/file" "file/android_pack" "file/ios_pack" "file/ini_pack"; do
    mkdir -p "$d"
done

[ ! -f "$SRC_DIR/data/friendb/friends.friendb" ] && touch "$SRC_DIR/data/friendb/friends.friendb"
[ -f "$ENV_DIR/ip2region.xdb" ] && cp -n "$ENV_DIR/ip2region.xdb" "$SRC_DIR/data/ip2region/ip2region.xdb" 2>/dev/null || true

log_info "运行时目录就绪"

# ============================================================
# 第三步: 生成 overlay.json (服务器路径)
# ============================================================
echo -e "${CYAN}[3/6] 生成 overlay 补丁映射...${NC}"

cat > "$PATCH_DIR/overlay.json" << EOFOVERLAY
{
  "Replace": {
    "$SRC_DIR/apps/server/main.go": "$PATCH_DIR/apps/server/main.go",
    "$SRC_DIR/plugin/plugin.http.server.go": "$PATCH_DIR/plugin/plugin.http.server.go",
    "$SRC_DIR/plugin/plugin.yidun.check.go": "$PATCH_DIR/plugin/plugin.yidun.check.go",
    "$SRC_DIR/apps/server/ctrler/ctrler.custservice.api.go": "$PATCH_DIR/apps/server/ctrler/ctrler.custservice.api.go"
  }
}
EOFOVERLAY

log_info "overlay.json 已生成 (4 个补丁文件)"

# ============================================================
# 第四步: 构建前端
# ============================================================
echo -e "${CYAN}[4/6] 构建前端...${NC}"

if [ -d "$WEB_DIR" ]; then
    # 应用前端补丁 (.web/setting.ts)
    WEB_PATCH="$PATCH_DIR/.web"
    if [ -d "$WEB_PATCH" ]; then
        cp -rf "$WEB_PATCH"/* "$WEB_DIR"/ 2>/dev/null || true
        log_info "前端补丁已应用"
    fi

    cd "$WEB_DIR"
    if [ ! -d "node_modules" ]; then
        log_info "安装前端依赖 (npm install)..."
        npm install --legacy-peer-deps 2>&1 | tail -1
    fi

    log_info "构建前端 (npm run build)..."
    npm run build 2>&1 | tail -3

    # 拷贝构建产物到 Go 后端 html 目录
    BUILD_OUTPUT="$WEB_DIR/dist"
    HTML_DIR="$SRC_DIR/apps/server/html"
    if [ -d "$BUILD_OUTPUT" ]; then
        rm -rf "$HTML_DIR"
        cp -r "$BUILD_OUTPUT" "$HTML_DIR"
        log_info "前端构建完成 → $HTML_DIR"
    else
        log_warn "前端构建产物未找到, Go 后端将无前端页面"
    fi
else
    log_warn "前端目录不存在: $WEB_DIR, 跳过前端构建"
fi

# ============================================================
# 第五步: 编译 Go 后端
# ============================================================
echo -e "${CYAN}[5/6] 编译 Go 后端...${NC}"

cd "$SRC_DIR"

export GOPROXY=https://goproxy.cn,direct

log_info "下载 Go 依赖..."
go mod download 2>&1 | tail -3

log_info "编译中 (go build -overlay)..."
go build -overlay="$PATCH_DIR/overlay.json" -o "$BASE_DIR/ymlink-q2-server" ./apps/server/

if [ -f "$BASE_DIR/ymlink-q2-server" ]; then
    log_info "编译成功 → $BASE_DIR/ymlink-q2-server"
else
    log_error "编译失败!"
fi

# ============================================================
# 第六步: 创建 systemd 服务
# ============================================================
echo -e "${CYAN}[6/6] 配置 systemd 服务...${NC}"

# 获取服务器公网 IP (用于显示)
PUBLIC_IP=$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || echo "YOUR_SERVER_IP")

cat > /etc/systemd/system/ymlink-q2.service << EOFSVC
[Unit]
Description=YMLink-Q2 Server
After=network.target mongod.service influxdb.service
Wants=mongod.service influxdb.service

[Service]
Type=simple
WorkingDirectory=$SRC_DIR
ExecStart=$BASE_DIR/ymlink-q2-server
Restart=always
RestartSec=5

# 环境变量
Environment=SYSTEM_MODE=debug
Environment=SYSTEM_LOG_LEVEL=debug
Environment=SYSTEM_MEM_LIMIT=0
Environment=SYSTEM_CPU_LIMIT=4
Environment=SYSTEM_ROBOT_THREAD_LIMIT=1
Environment=SYSTEM_ROBOT_MESSAGE_LIMIT=100
Environment=SYSTEM_DBLOAD=true

Environment=HTTP_SERVER1_URL=:8080
Environment=HTTP_SERVER1_MODE=debug
Environment=HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret
Environment=HTTP_SERVER1_AUTH_TIMEOUT=604800
Environment=HTTP_SERVER1_HTML_PATH=./apps/server/html

Environment=MONGO1_URL=localhost:27017/ymlink_q2?authSource=admin
Environment=MONGO1_DATABASE=ymlink_q2
Environment=MONGO1_USERNAME=admin
Environment=MONGO1_PASSWORD=admin

Environment=INFLUX1_URL=http://localhost:8086
Environment=INFLUX1_ORG=ymlink
Environment=INFLUX1_TOKEN=ymlink-influx-token

Environment=FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
Environment=FRIENDB1_TOTAL=100000
Environment=IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb

Environment=DRIVE_MAPPING=server1#iOSQQ#8.9.80#http://8.130.31.166:8098
Environment=DRIVE_TIMEOUT=30
Environment=DRIVE_MAX_CONN=50
Environment=MOBILE_PORT=16100

[Install]
WantedBy=multi-user.target
EOFSVC

systemctl daemon-reload
systemctl enable ymlink-q2
systemctl restart ymlink-q2

sleep 3

# ============================================================
# 完成
# ============================================================
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║   部署完成!                                      ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}访问地址:${NC}  http://${PUBLIC_IP}:8080"
echo -e "  ${CYAN}登录账号:${NC}  admin"
echo -e "  ${CYAN}登录密码:${NC}  a12345677"
echo ""
echo -e "  ${CYAN}常用命令:${NC}"
echo -e "    sudo systemctl status ymlink-q2    # 查看状态"
echo -e "    sudo systemctl restart ymlink-q2   # 重启服务"
echo -e "    sudo systemctl stop ymlink-q2      # 停止服务"
echo -e "    sudo journalctl -u ymlink-q2 -f    # 查看实时日志"
echo ""

# 检查服务状态
if systemctl is-active --quiet ymlink-q2; then
    log_info "服务运行中 ✓"
else
    log_warn "服务可能未启动成功, 查看日志: journalctl -u ymlink-q2 -n 50"
fi

# 防火墙 (如果开了 ufw)
if command -v ufw &>/dev/null && ufw status | grep -q "active"; then
    ufw allow 8080/tcp 2>/dev/null || true
    log_info "防火墙已放行 8080 端口"
fi

echo ""
