#!/bin/bash
# q2-env deploy script
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step()  { echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${CYAN}  $1${NC}"; echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

TOTAL=5
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(dirname "$SCRIPT_DIR")"
ENV_DIR="$SCRIPT_DIR"
BINARY_SRC="$ENV_DIR/ymlink-server"
BINARY="$BASE_DIR/q2-server"
WORK_DIR="$BASE_DIR/server-data"

[ "$(id -u)" -ne 0 ] && { log_error "root required"; exit 1; }

echo ""
echo "=========================================="
echo "  deploy - $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="

# [1] deps
log_step "[1/$TOTAL] deps"
export DEBIAN_FRONTEND=noninteractive
PKGS=""
command -v curl &>/dev/null || PKGS="$PKGS curl"
command -v jq   &>/dev/null || PKGS="$PKGS jq"
[ -n "$PKGS" ] && { apt-get update -qq; apt-get install -y -qq $PKGS; }

if command -v docker &>/dev/null; then
    log_info "docker $(docker --version | awk '{print $3}' | tr -d ',')"
else
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker && systemctl start docker
fi
docker compose version &>/dev/null 2>&1 || apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
log_info "deps ok"

# [2] binary
log_step "[2/$TOTAL] binary"
[ ! -f "$BINARY_SRC" ] && { log_error "binary not found: $BINARY_SRC"; exit 1; }
cp -f "$BINARY_SRC" "$BINARY" && chmod +x "$BINARY"
log_info "server: $(du -h "$BINARY" | awk '{print $1}')"

# [3] docker services
log_step "[3/$TOTAL] docker services"
cd "$ENV_DIR"
COMPOSE="docker compose"
$COMPOSE version &>/dev/null 2>&1 || COMPOSE="docker-compose"
$COMPOSE up -d mongodb influxdb

echo -n "  waiting mongo"
for i in $(seq 1 60); do
    docker exec q2-mongo mongosh --quiet --eval "db.runCommand({ping:1})" \
        -u admin -p admin --authenticationDatabase admin &>/dev/null 2>&1 && { echo ""; log_info "mongo ok"; break; }
    echo -n "."; sleep 2
    [ $i -eq 60 ] && { echo ""; log_error "mongo timeout"; exit 1; }
done

echo -n "  waiting influx"
for i in $(seq 1 40); do
    curl -sf http://localhost:8086/health 2>/dev/null | grep -q '"status":"pass"' && { echo ""; log_info "influx ok"; break; }
    echo -n "."; sleep 2
    [ $i -eq 40 ] && { echo ""; log_error "influx timeout"; exit 1; }
done
docker exec q2-influx influx bucket create --name history --org q2org --retention 8760h --token q2-influx-token 2>/dev/null || true

# [4] runtime
log_step "[4/$TOTAL] runtime"
mkdir -p "$WORK_DIR" && cd "$WORK_DIR"
for d in data/friendb data/ip2region log file/task file/login file/material file/message file/usedb file/qzonedb file/materialdb file/realinfodb file/android_pack file/ios_pack file/ini_pack html; do
    mkdir -p "$d"
done
if [ ! -f "data/ip2region/ip2region.xdb" ]; then
    [ -f "$ENV_DIR/ip2region.xdb" ] && cp "$ENV_DIR/ip2region.xdb" data/ip2region/ip2region.xdb || \
    curl -fsSL "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region.xdb" -o data/ip2region/ip2region.xdb
fi
[ ! -f "data/friendb/friends.friendb" ] && touch data/friendb/friends.friendb
MAP=$(sysctl -n vm.max_map_count 2>/dev/null || echo "0")
[ "$MAP" -lt 262144 ] && { sysctl -w vm.max_map_count=262144 >/dev/null; grep -q 'vm.max_map_count' /etc/sysctl.conf 2>/dev/null || echo 'vm.max_map_count=262144' >> /etc/sysctl.conf; }
log_info "runtime ok"

# [5] start
log_step "[5/$TOTAL] start"

cat > "$BASE_DIR/start.sh" << STARTEOF
#!/bin/bash
BASE="\$(cd "\$(dirname "\$0")" && pwd)"
BINARY="\$BASE/q2-server"
WORK="\$BASE/server-data"
cd "\$WORK"
export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_THREAD_LIMIT=1 SYSTEM_MSG_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=q2-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./html
export MONGO1_URL='localhost:27017/q2_db?authSource=admin' MONGO1_DATABASE=q2_db
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=q2org INFLUX1_TOKEN=q2-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING='server1#client#8.9.80#http://8.130.31.166:8098'
export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100
pkill -f q2-server 2>/dev/null || true; sleep 1
nohup "\$BINARY" > "\$BASE/server.log" 2>&1 &
echo \$!
STARTEOF
chmod +x "$BASE_DIR/start.sh"

cat > /etc/systemd/system/q2-server.service << SVCEOF
[Unit]
Description=Q2 Server
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
Environment=SYSTEM_THREAD_LIMIT=1
Environment=SYSTEM_MSG_LIMIT=100
Environment=SYSTEM_DBLOAD=true
Environment=HTTP_SERVER1_URL=:8080
Environment=HTTP_SERVER1_MODE=debug
Environment=HTTP_SERVER1_AUTH_PASSWORD=q2-jwt-secret
Environment=HTTP_SERVER1_AUTH_TIMEOUT=604800
Environment=HTTP_SERVER1_HTML_PATH=./html
Environment=MONGO1_URL=localhost:27017/q2_db?authSource=admin
Environment=MONGO1_DATABASE=q2_db
Environment=MONGO1_USERNAME=admin
Environment=MONGO1_PASSWORD=admin
Environment=INFLUX1_URL=http://localhost:8086
Environment=INFLUX1_ORG=q2org
Environment=INFLUX1_TOKEN=q2-influx-token
Environment=FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
Environment=FRIENDB1_TOTAL=100000
Environment=IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
Environment=DRIVE_MAPPING=server1#client#8.9.80#http://8.130.31.166:8098
Environment=DRIVE_TIMEOUT=30
Environment=DRIVE_MAX_CONN=50
Environment=MOBILE_PORT=16100
[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable q2-server.service >/dev/null 2>&1

NEW_PID=$(bash "$BASE_DIR/start.sh")
log_info "PID: $NEW_PID"

echo -n "  waiting"
for i in $(seq 1 30); do
    CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:8080/" 2>/dev/null)
    [ "$CODE" = "200" ] || [ "$CODE" = "401" ] || [ "$CODE" = "302" ] && { echo ""; log_info "ready ($CODE)"; break; }
    echo -n "."; sleep 2
    [ $i -eq 30 ] && { echo ""; log_warn "check: tail -f $BASE_DIR/server.log"; }
done

IP=$(hostname -I | awk '{print $1}')
echo ""
echo -e "${GREEN}done${NC}  http://${IP}:8080"
echo -e "  log:     tail -f $BASE_DIR/server.log"
echo -e "  service: systemctl start|stop|restart q2-server"
echo -e "  update:  cd $ENV_DIR && git pull && bash ubuntu-deploy.sh"
echo ""
