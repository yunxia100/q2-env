#!/bin/bash
# ============================================================
#  YMLink-Q2 Ubuntu 一键部署
#
#  仓库只包含: 编译好的二进制 + 脚本 + 配置，无源码。
#
#  目录结构:
#    /root/q2/
#    ├── q2-env/                 # 本仓库 (git clone)
#    │   ├── ymlink-server       # 预编译 Linux 二进制 (27MB)
#    │   ├── docker-compose.yml  # MongoDB + InfluxDB
#    │   ├── ip2region.xdb       # IP归属地库
#    │   └── ubuntu-deploy.sh    # 本脚本
#    └── (其他你需要的文件)
#
#  用法:
#    cd /root/q2
#    git clone https://github.com/yunxia100/q2-env.git
#    bash q2-env/ubuntu-deploy.sh
# ============================================================

set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

TOTAL=5

# ========================== 路径 ==========================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"
ENV_DIR="$SCRIPT_DIR"

BINARY_SRC="$ENV_DIR/ymlink-server"
BINARY="$BASE_DIR/ymlink-server"
WORK_DIR="$BASE_DIR/server-data"

if [ "$(id -u)" -ne 0 ]; then log_error "请用 root 运行"; exit 1; fi

echo ""
echo "=========================================================="
echo "  YMLink-Q2 一键部署"
echo "  $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================================="

# ========================== [1] 系统依赖 + Docker ==========================
log_step "[1/$TOTAL] 安装依赖"

export DEBIAN_FRONTEND=noninteractive
PKGS=""
command -v curl &>/dev/null || PKGS="$PKGS curl"
command -v jq   &>/dev/null || PKGS="$PKGS jq"
[ -n "$PKGS" ] && { apt-get update -qq; apt-get install -y -qq $PKGS; }

if command -v docker &>/dev/null; then
    log_info "Docker: $(docker --version | awk '{print $3}' | tr -d ',')"
else
    log_warn "安装 Docker ..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
    log_info "Docker 安装完成"
fi
docker compose version &>/dev/null 2>&1 || apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
log_info "依赖就绪"

# ========================== [2] 检查二进制 ==========================
log_step "[2/$TOTAL] 检查二进制"

if [ ! -f "$BINARY_SRC" ]; then
    log_error "二进制不存在: $BINARY_SRC"
    log_error "请确认仓库完整: git clone https://github.com/yunxia100/q2-env.git"
    exit 1
fi
cp -f "$BINARY_SRC" "$BINARY" && chmod +x "$BINARY"
log_info "ymlink-server: $(du -h "$BINARY" | awk '{print $1}')"

# ========================== [3] Docker 服务 ==========================
log_step "[3/$TOTAL] 启动 MongoDB + InfluxDB"

cd "$ENV_DIR"
COMPOSE="docker compose"
$COMPOSE version &>/dev/null 2>&1 || COMPOSE="docker-compose"
$COMPOSE up -d mongodb influxdb

echo -n "  等待 MongoDB"
for i in $(seq 1 60); do
    docker exec ymlink-mongo mongosh --quiet --eval "db.runCommand({ping:1})" \
        -u admin -p admin --authenticationDatabase admin &>/dev/null 2>&1 && {
        echo ""; log_info "MongoDB 就绪"; break
    }
    echo -n "."; sleep 2
    [ $i -eq 60 ] && { echo ""; log_error "MongoDB 超时"; exit 1; }
done

echo -n "  等待 InfluxDB"
for i in $(seq 1 40); do
    curl -sf http://localhost:8086/health 2>/dev/null | grep -q '"status":"pass"' && {
        echo ""; log_info "InfluxDB 就绪"; break
    }
    echo -n "."; sleep 2
    [ $i -eq 40 ] && { echo ""; log_error "InfluxDB 超时"; exit 1; }
done
docker exec ymlink-influx influx bucket create --name history --org ymlink --retention 8760h --token ymlink-influx-token 2>/dev/null || true

# ========================== [4] 运行时数据 ==========================
log_step "[4/$TOTAL] 准备运行时目录"

mkdir -p "$WORK_DIR"
cd "$WORK_DIR"

for d in data/friendb data/ip2region log file/task file/login file/material file/message file/usedb file/qzonedb file/materialdb file/realinfodb file/android_pack file/ios_pack file/ini_pack html; do
    mkdir -p "$d"
done

if [ ! -f "data/ip2region/ip2region.xdb" ]; then
    if [ -f "$ENV_DIR/ip2region.xdb" ]; then
        cp "$ENV_DIR/ip2region.xdb" data/ip2region/ip2region.xdb
    else
        curl -fsSL "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region.xdb" \
            -o data/ip2region/ip2region.xdb || { log_error "下载 ip2region.xdb 失败"; exit 1; }
    fi
fi
[ ! -f "data/friendb/friends.friendb" ] && touch data/friendb/friends.friendb

# 内核参数
MAP=$(sysctl -n vm.max_map_count 2>/dev/null || echo "0")
[ "$MAP" -lt 262144 ] && { sysctl -w vm.max_map_count=262144 >/dev/null; grep -q 'vm.max_map_count' /etc/sysctl.conf 2>/dev/null || echo 'vm.max_map_count=262144' >> /etc/sysctl.conf; }

log_info "运行时目录: $WORK_DIR"

# ========================== [5] 启动 ==========================
log_step "[5/$TOTAL] 启动服务"

# ---------- start.sh ----------
cat > "$BASE_DIR/start.sh" << STARTEOF
#!/bin/bash
BASE="\$(cd "\$(dirname "\$0")" && pwd)"
BINARY="\$BASE/ymlink-server"
WORK="\$BASE/server-data"
cd "\$WORK"

export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_ROBOT_THREAD_LIMIT=1 SYSTEM_ROBOT_MESSAGE_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./html
export MONGO1_URL='localhost:27017/ymlink_q2?authSource=admin' MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=ymlink INFLUX1_TOKEN=ymlink-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING='server1#iOSQQ#8.9.80#http://8.130.31.166:8098'
export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

pkill -f ymlink-server 2>/dev/null || true; sleep 1
nohup "\$BINARY" > "\$BASE/ymlink.log" 2>&1 &
echo \$!
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
WorkingDirectory=$WORK_DIR
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
Environment=HTTP_SERVER1_HTML_PATH=./html
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
log_info "systemd 开机自启已注册"

NEW_PID=$(bash "$BASE_DIR/start.sh")
log_info "PID: $NEW_PID"

echo -n "  等待就绪"
for i in $(seq 1 30); do
    CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:8080/" 2>/dev/null)
    [ "$CODE" = "200" ] || [ "$CODE" = "401" ] || [ "$CODE" = "302" ] && { echo ""; log_info "HTTP 就绪 ($CODE)"; break; }
    echo -n "."; sleep 2
    [ $i -eq 30 ] && { echo ""; log_warn "检查日志: tail -f $BASE_DIR/ymlink.log"; }
done

# ========================== 完成 ==========================
IP=$(hostname -I | awk '{print $1}')

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  部署完成！                                      ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "  ${CYAN}后台:${NC}  http://${IP}:8080"
echo -e "  ${CYAN}日志:${NC}  tail -f $BASE_DIR/ymlink.log"
echo ""
echo -e "  systemctl start|stop|restart|status ymlink"
echo -e "  更新: cd $ENV_DIR && git pull && bash ubuntu-deploy.sh"
echo ""
