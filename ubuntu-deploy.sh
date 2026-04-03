#!/bin/bash
# ============================================================
#  YMLink-Q2 Ubuntu 一键部署脚本
#
#  仓库里已包含编译好的 Linux 二进制 (ymlink-server)，
#  新服务器不需要安装 Go，不需要编译，拉取即可运行。
#
#  目录结构:
#    /root/q2/
#    ├── ymlink-q2-new-master/   # 后端源码 (你手动放入)
#    ├── q2-env/                 # 本仓库 (git clone)
#    │   ├── ymlink-server       # 预编译的 Linux 二进制
#    │   ├── patches/            # Go 补丁源码 (备查)
#    │   ├── docker-compose.yml  # MongoDB + InfluxDB
#    │   └── ip2region.xdb
#    └── ymlink-q2-ui-main/     # 前端源码 (可选)
#
#  用法:
#    cd /root/q2
#    git clone https://github.com/yunxia100/q2-env.git
#    bash q2-env/ubuntu-deploy.sh
# ============================================================

set -euo pipefail

# ========================== 颜色 ==========================
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

TOTAL=7

# ========================== 自动检测路径 ==========================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

ENV_DIR="$SCRIPT_DIR"
SRC_DIR="$BASE_DIR/ymlink-q2-new-master"
UI_DIR="$BASE_DIR/ymlink-q2-ui-main"
BINARY_SRC="$ENV_DIR/ymlink-server"
BINARY="$BASE_DIR/ymlink-server"
WEB_DIR="$SRC_DIR/.web"

DRIVE_HOST="8.130.31.166"
DRIVE_PORT="8098"

# ========================== 检测 ==========================
if [ "$(id -u)" -ne 0 ]; then
    log_error "请使用 root 用户运行"; exit 1
fi

echo ""
echo "=========================================================="
echo "  YMLink-Q2 Ubuntu 一键部署"
echo "  时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "  目录: $BASE_DIR"
echo "=========================================================="

# ========================== [1] 安装系统依赖 ==========================
log_step "[1/$TOTAL] 安装系统依赖"

export DEBIAN_FRONTEND=noninteractive

PKGS=""
command -v curl    &>/dev/null || PKGS="$PKGS curl"
command -v wget    &>/dev/null || PKGS="$PKGS wget"
command -v jq      &>/dev/null || PKGS="$PKGS jq"
command -v python3 &>/dev/null || PKGS="$PKGS python3"
command -v sshpass &>/dev/null || PKGS="$PKGS sshpass"

if [ -n "$PKGS" ]; then
    apt-get update -qq && apt-get install -y -qq $PKGS
fi
log_info "系统包就绪"

# ========================== [2] 安装 Docker ==========================
log_step "[2/$TOTAL] 安装 Docker"

if command -v docker &>/dev/null; then
    log_info "Docker 已安装: $(docker --version | awk '{print $3}' | tr -d ',')"
else
    log_warn "安装 Docker ..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
    log_info "Docker 安装完成"
fi

# docker compose plugin
docker compose version &>/dev/null 2>&1 || {
    apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
}
log_info "Docker 就绪"

# ========================== [3] 检查文件 ==========================
log_step "[3/$TOTAL] 检查文件"

# 二进制
if [ ! -f "$BINARY_SRC" ]; then
    log_error "预编译二进制不存在: $BINARY_SRC"
    log_error "请确认仓库完整: git clone https://github.com/yunxia100/q2-env.git"
    exit 1
fi

# 复制二进制到 /root/q2/
cp -f "$BINARY_SRC" "$BINARY"
chmod +x "$BINARY"
log_info "ymlink-server: $(du -h "$BINARY" | awk '{print $1}') ($(file "$BINARY" | grep -oP 'ELF.*?,' | head -1))"

# 后端源码 (运行时需要 html/ 和 data/ 目录结构)
if [ ! -d "$SRC_DIR" ]; then
    log_error "后端源码不存在: $SRC_DIR"
    log_error "请将 ymlink-q2-new-master/ 放到 $BASE_DIR/"
    exit 1
fi
log_info "后端源码: $SRC_DIR"

# ========================== [4] Docker 服务 ==========================
log_step "[4/$TOTAL] 启动 Docker 服务 (MongoDB + InfluxDB)"

cd "$ENV_DIR"
COMPOSE="docker compose"
$COMPOSE version &>/dev/null 2>&1 || COMPOSE="docker-compose"

$COMPOSE up -d mongodb influxdb

# 等 MongoDB
echo -n "  等待 MongoDB"
for i in $(seq 1 60); do
    docker exec ymlink-mongo mongosh --quiet --eval "db.runCommand({ping:1})" \
        -u admin -p admin --authenticationDatabase admin &>/dev/null 2>&1 && {
        echo ""; log_info "MongoDB 就绪"; break
    }
    echo -n "."; sleep 2
    [ $i -eq 60 ] && { echo ""; log_error "MongoDB 超时"; exit 1; }
done

# 等 InfluxDB
echo -n "  等待 InfluxDB"
for i in $(seq 1 40); do
    curl -sf http://localhost:8086/health 2>/dev/null | grep -q '"status":"pass"' && {
        echo ""; log_info "InfluxDB 就绪"; break
    }
    echo -n "."; sleep 2
    [ $i -eq 40 ] && { echo ""; log_error "InfluxDB 超时"; exit 1; }
done

docker exec ymlink-influx influx bucket create --name history --org ymlink --retention 8760h --token ymlink-influx-token 2>/dev/null || true

# ========================== [5] 运行时环境 ==========================
log_step "[5/$TOTAL] 准备运行时环境"

cd "$SRC_DIR"

for d in data/friendb data/ip2region log file/task file/login file/material file/message file/usedb file/qzonedb file/materialdb file/realinfodb file/android_pack file/ios_pack file/ini_pack apps/server/html; do
    mkdir -p "$d"
done

# ip2region.xdb
if [ ! -f "data/ip2region/ip2region.xdb" ]; then
    if [ -f "$ENV_DIR/ip2region.xdb" ]; then
        cp "$ENV_DIR/ip2region.xdb" data/ip2region/ip2region.xdb
    else
        wget -q "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region.xdb" \
            -O data/ip2region/ip2region.xdb || { log_error "下载 ip2region.xdb 失败"; exit 1; }
    fi
fi
log_info "ip2region.xdb 就绪"

[ ! -f "data/friendb/friends.friendb" ] && touch data/friendb/friends.friendb

# 内核参数
CURRENT_MAP=$(sysctl -n vm.max_map_count 2>/dev/null || echo "0")
if [ "$CURRENT_MAP" -lt 262144 ]; then
    sysctl -w vm.max_map_count=262144 >/dev/null
    grep -q 'vm.max_map_count' /etc/sysctl.conf 2>/dev/null || echo 'vm.max_map_count=262144' >> /etc/sysctl.conf
fi

log_info "运行时目录就绪"

# ========================== [6] 编译前端 (可选) ==========================
log_step "[6/$TOTAL] 前端"

# 如果有 Node.js 和前端源码，编译前端
ACTUAL_WEB=""
[ -d "$WEB_DIR" ] && [ -f "$WEB_DIR/package.json" ] && ACTUAL_WEB="$WEB_DIR"
[ -z "$ACTUAL_WEB" ] && [ -d "$UI_DIR" ] && [ -f "$UI_DIR/package.json" ] && ACTUAL_WEB="$UI_DIR"

if [ -n "$ACTUAL_WEB" ] && command -v npm &>/dev/null; then
    WEB_PATCH_DIR="$ENV_DIR/patches/.web"
    BACKUP_DIR="/tmp/ymlink-web-backup"
    PATCHED_FILES=()

    if [ -d "$WEB_PATCH_DIR" ]; then
        mkdir -p "$BACKUP_DIR"
        while IFS= read -r pf; do
            rel="${pf#$WEB_PATCH_DIR/}"
            orig="$ACTUAL_WEB/$rel"; bak="$BACKUP_DIR/$rel"
            [ -f "$orig" ] && { mkdir -p "$(dirname "$bak")"; cp -p "$orig" "$bak"; }
            mkdir -p "$(dirname "$orig")"; cp "$pf" "$orig"
            PATCHED_FILES+=("$rel")
        done < <(find "$WEB_PATCH_DIR" -type f ! -name '.DS_Store')
        log_info "应用 ${#PATCHED_FILES[@]} 个前端补丁"
    fi

    cd "$ACTUAL_WEB"
    [ ! -d "node_modules" ] && npm install --legacy-peer-deps 2>&1 | tail -3
    npm run build 2>&1 | tail -5

    # 还原
    for rel in "${PATCHED_FILES[@]}"; do
        orig="$ACTUAL_WEB/$rel"; bak="$BACKUP_DIR/$rel"
        [ -f "$bak" ] && cp -p "$bak" "$orig" || rm -f "$orig"
    done
    rm -rf "$BACKUP_DIR"

    DIST=""
    [ -d "$ACTUAL_WEB/dist" ] && DIST="$ACTUAL_WEB/dist"
    if [ -n "$DIST" ]; then
        mkdir -p "$SRC_DIR/apps/server/html"
        cp -r "$DIST/"* "$SRC_DIR/apps/server/html/"
        log_info "前端编译完成 → html/"
    fi
else
    if [ -f "$SRC_DIR/apps/server/html/index.html" ]; then
        log_info "使用已有 html/"
    else
        log_warn "无前端 / 无 Node.js，跳过。后端 patchJS 仍生效。"
        log_warn "如需前端: apt install nodejs npm && bash ubuntu-deploy.sh"
    fi
fi

# ========================== [7] 启动服务 ==========================
log_step "[7/$TOTAL] 启动服务"

cd "$SRC_DIR"

# ---------- start.sh ----------
cat > "$BASE_DIR/start.sh" << 'STARTEOF'
#!/bin/bash
BASE_DIR="$(cd "$(dirname "$0")" && pwd)"
SRC_DIR="$BASE_DIR/ymlink-q2-new-master"
BINARY="$BASE_DIR/ymlink-server"
cd "$SRC_DIR"

export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_ROBOT_THREAD_LIMIT=1 SYSTEM_ROBOT_MESSAGE_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./apps/server/html
export MONGO1_URL='localhost:27017/ymlink_q2?authSource=admin' MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=ymlink INFLUX1_TOKEN=ymlink-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING='server1#iOSQQ#8.9.80#http://8.130.31.166:8098'
export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

pkill -f ymlink-server 2>/dev/null || true; sleep 1
nohup "$BINARY" > "$BASE_DIR/ymlink.log" 2>&1 &
echo $!
STARTEOF
chmod +x "$BASE_DIR/start.sh"

# ---------- systemd ----------
cat > /etc/systemd/system/ymlink.service << SVCEOF
[Unit]
Description=YMLink-Q2 Server
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=root
WorkingDirectory=$SRC_DIR
ExecStartPre=/bin/bash -c 'cd $ENV_DIR && docker compose up -d mongodb influxdb 2>/dev/null || docker-compose up -d mongodb influxdb; sleep 5'
ExecStart=$BINARY
Restart=on-failure
RestartSec=5
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
SVCEOF

systemctl daemon-reload
systemctl enable ymlink.service >/dev/null 2>&1
log_info "systemd 服务已注册 (开机自启)"

NEW_PID=$(bash "$BASE_DIR/start.sh")
log_info "服务已启动 PID: $NEW_PID"

echo -n "  等待服务就绪"
for i in $(seq 1 30); do
    HTTP_CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:8080/" 2>/dev/null)
    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "302" ]; then
        echo ""; log_info "HTTP 就绪 ($HTTP_CODE)"; break
    fi
    echo -n "."; sleep 2
    [ $i -eq 30 ] && { echo ""; log_warn "检查日志: tail -f $BASE_DIR/ymlink.log"; }
done

# ========================== 完成 ==========================
SERVER_IP=$(hostname -I | awk '{print $1}')

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  YMLink-Q2 部署完成！                                ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}后台地址:${NC}  http://${SERVER_IP}:8080"
echo -e "  ${CYAN}目录:${NC}      $BASE_DIR"
echo -e "  ${CYAN}日志:${NC}      tail -f $BASE_DIR/ymlink.log"
echo ""
echo -e "  ${YELLOW}常用命令:${NC}"
echo -e "    systemctl start|stop|restart|status ymlink"
echo -e "    journalctl -u ymlink -f"
echo ""
echo -e "  ${YELLOW}更新:${NC}  cd $ENV_DIR && git pull && bash ubuntu-deploy.sh"
echo ""
