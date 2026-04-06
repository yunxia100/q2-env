#!/bin/bash
# ============================================================
#  YMLink-Q2 一键部署脚本（司法鉴定版）
#
#  用法:
#    全量部署:  bash server-deploy.sh
#    仅编译:    bash server-deploy.sh build
#    仅启动:    bash server-deploy.sh start
#    仅更新:    bash server-deploy.sh update
#    停止服务:  bash server-deploy.sh stop
#
#  原始源码不可修改，补丁通过 Go overlay 机制注入编译
# ============================================================

# ************************************************************
# *                    配置区（演示时修改）                     *
# ************************************************************

# ---- 原始源码目录（司法鉴定用，部署后不可修改）----
SRC_GO="/root/q2/ymlink-q2-new-master"         # Go 后端原始源码
SRC_UI="/root/q2/ymlink-q2-ui-main"             # Vue 前端原始源码

# ---- 工作目录 ----
BASE="/root/q2"                                 # 项目根目录
DATA="$BASE/server-data"                        # 运行时数据目录
HTTP_PORT="8080"                                # 服务端口

# ---- 补丁仓库（git 拉取）----
GIT_REPO="https://github.com/yunxia100/q2-env.git"
ENV="/root/env"                                 # 补丁仓库存放目录

# ---- 数据库配置 ----
MONGO_USER="admin"
MONGO_PASS="admin"
MONGO_DB="q2_db"
INFLUX_ORG="q2org"
INFLUX_TOKEN="q2-influx-token"

# ************************************************************
# *                    以下无需修改                            *
# ************************************************************

PATCHES="$ENV/patches"
BINARY="$ENV/q2-env-patch"
WEB_DIST="$ENV/web-env"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_ok()   { echo -e "  ${GREEN}[OK]${NC}  $1"; }
log_warn() { echo -e "  ${YELLOW}[!]${NC}   $1"; }
log_err()  { echo -e "  ${RED}[ERR]${NC} $1"; exit 1; }
step()     { echo -e "\n${CYAN}==> $1${NC}"; }

# ============================================================
# [1] 检查源码
# ============================================================
do_check_src() {
    step "检查原始源码目录"
    [ ! -f "$SRC_GO/go.mod" ]     && log_err "Go 源码无效: $SRC_GO"
    [ ! -f "$SRC_UI/package.json" ] && log_err "前端源码无效: $SRC_UI"
    log_ok "Go 源码: $SRC_GO"
    log_ok "Vue 源码: $SRC_UI"
}

# ============================================================
# [2] 安装依赖
# ============================================================
do_install_deps() {
    step "安装系统依赖"
    export DEBIAN_FRONTEND=noninteractive

    # 基础工具
    for cmd in git curl rsync wget jq; do
        command -v $cmd &>/dev/null || { apt-get update -qq && apt-get install -y -qq git curl jq rsync wget; break; }
    done
    log_ok "基础工具就绪"

    # Docker（仅 InfluxDB 需要）
    if ! command -v docker &>/dev/null; then
        apt-get install -y -qq docker.io 2>/dev/null || (curl -fsSL https://get.docker.com | sh)
        systemctl enable docker && systemctl start docker
    fi
    # 中国 Docker 镜像加速
    if ! grep -q "registry-mirrors" /etc/docker/daemon.json 2>/dev/null; then
        mkdir -p /etc/docker
        cat > /etc/docker/daemon.json <<'MIRRORS'
{
  "registry-mirrors": [
    "https://dockerpull.org",
    "https://docker.rainbond.cc",
    "https://docker.1ms.run",
    "https://docker.xuanyuan.me"
  ]
}
MIRRORS
        systemctl restart docker && sleep 3
    fi
    log_ok "Docker 就绪（中国镜像加速）"

    # Go
    export PATH=$PATH:/usr/local/go/bin
    if ! command -v go &>/dev/null; then
        GO_VER="1.21.13"
        wget -q "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -O /tmp/go.tar.gz || \
        curl -fsSL "https://go.dev/dl/go${GO_VER}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
        rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && rm -f /tmp/go.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
        go version &>/dev/null || log_err "Go 安装失败"
    fi
    log_ok "Go: $(go version | awk '{print $3}')"
    export GOPATH=/root/go GOPROXY=https://goproxy.cn,direct

    # Node.js
    if ! command -v node &>/dev/null; then
        curl -fsSL https://deb.nodesource.com/setup_18.x | bash - 2>/dev/null
        apt-get install -y -qq nodejs 2>/dev/null
        node -v &>/dev/null || log_err "Node.js 安装失败"
    fi
    log_ok "Node.js: $(node -v)"
}

# ============================================================
# [3] 拉取补丁仓库
# ============================================================
do_clone_repo() {
    step "拉取补丁仓库 → $ENV"
    if [ -d "$ENV/.git" ]; then
        cd "$ENV" && git pull
        log_ok "补丁仓库已更新"
    else
        rm -rf "$ENV"
        git clone "$GIT_REPO" "$ENV"
        log_ok "补丁仓库克隆完成"
    fi
    [ ! -d "$PATCHES" ] && log_err "补丁目录不存在: $PATCHES"
}

# ============================================================
# [4] 安装数据库
# ============================================================
do_install_db() {
    step "安装数据库"

    # --- MongoDB（apt 安装，中国比 Docker 快）---
    if ! command -v mongosh &>/dev/null && ! command -v mongod &>/dev/null; then
        curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | \
            gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg 2>/dev/null
        echo "deb [signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse" \
            > /etc/apt/sources.list.d/mongodb-org-7.0.list
        apt-get update -qq && apt-get install -y -qq mongodb-org
        log_ok "MongoDB 安装完成"
    fi

    # 启动 MongoDB
    if ! systemctl is-active mongod &>/dev/null; then
        systemctl enable mongod && systemctl start mongod
        sleep 3
    fi

    # 初始化 admin 用户
    mongosh admin --quiet --eval "
        if (db.getUser('$MONGO_USER') === null) {
            db.createUser({user:'$MONGO_USER', pwd:'$MONGO_PASS', roles:['root']});
            print('用户已创建');
        } else {
            print('用户已存在');
        }
    " 2>/dev/null || true
    log_ok "MongoDB 运行中 (端口 27017)"

    # --- InfluxDB（Docker，中国镜像加速）---
    if ! curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass; then
        docker rm -f q2-influx 2>/dev/null || true
        docker run -d --name q2-influx --restart always \
            -p 8086:8086 \
            -e DOCKER_INFLUXDB_INIT_MODE=setup \
            -e DOCKER_INFLUXDB_INIT_USERNAME=admin \
            -e DOCKER_INFLUXDB_INIT_PASSWORD=admin123 \
            -e DOCKER_INFLUXDB_INIT_ORG=$INFLUX_ORG \
            -e DOCKER_INFLUXDB_INIT_BUCKET=realtime \
            -e DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=$INFLUX_TOKEN \
            -v influxdb2-data:/var/lib/influxdb2 \
            influxdb:2.7

        echo -n "  等待 InfluxDB"
        for i in $(seq 1 40); do
            curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass && { echo ""; break; }
            echo -n "."; sleep 2
            [ $i -eq 40 ] && { echo ""; log_err "InfluxDB 启动超时"; }
        done
    fi

    # 创建 history bucket
    docker exec q2-influx influx bucket create \
        --name history --org $INFLUX_ORG --retention 8760h \
        --token $INFLUX_TOKEN 2>/dev/null || true
    log_ok "InfluxDB 运行中 (端口 8086)"
}

# ============================================================
# [5] Go overlay 编译
# ============================================================
do_build_go() {
    step "Go overlay 编译（原始源码不修改）"
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=/root/go GOPROXY=https://goproxy.cn,direct

    # 自动生成 overlay.json
    OVERLAY="$PATCHES/overlay.json"
    FIRST=true; COUNT=0
    {
        echo '{"Replace":{'
        while IFS= read -r pf; do
            rel="${pf#$PATCHES/}"; orig="$SRC_GO/$rel"
            [ "$FIRST" = true ] && FIRST=false || echo ','
            printf '"%s":"%s"' "$orig" "$pf"
            COUNT=$((COUNT + 1))
        done < <(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | sort)
        echo '}}'
    } > "$OVERLAY"
    log_ok "overlay.json: $COUNT 个补丁"

    pkill -f q2-env-patch 2>/dev/null; sleep 1
    cd "$SRC_GO"
    go build -overlay="$OVERLAY" -o "$BINARY" ./apps/server/
    chmod +x "$BINARY"
    log_ok "编译完成: $BINARY ($(ls -lh "$BINARY" | awk '{print $5}'))"
}

# ============================================================
# [6] 前端编译
# ============================================================
do_build_frontend() {
    step "前端编译（补丁覆盖 → vite build）"
    WEB_PATCHES="$PATCHES/.web"
    if [ -d "$WEB_PATCHES" ]; then
        TEMP="$BASE/.ui-build-temp"
        rm -rf "$TEMP"
        cp -a "$SRC_UI" "$TEMP"
        rsync -a --exclude='*.md' "$WEB_PATCHES/" "$TEMP/"
        log_ok "前端补丁: $(find "$WEB_PATCHES" -type f | wc -l) 个文件"
        cd "$TEMP"
        npm install --prefer-offline 2>/dev/null || npm install
        npx vite build --outDir "$WEB_DIST"
        rm -rf "$TEMP"
        log_ok "vite build 完成（原始源码未修改）"
    else
        log_warn "无前端补丁，使用已有 web-env/"
    fi
}

# ============================================================
# [7] 初始化运行目录 + 软链接
# ============================================================
do_init_data() {
    step "初始化运行时目录"
    mkdir -p "$DATA" && cd "$DATA"
    for d in data/friendb data/ip2region log html \
        file/{task,login,material,materialdb,message,usedb,qzonedb,realinfodb,android_pack,ios_pack,ini_pack}; do
        mkdir -p "$d"
    done

    # ip2region 数据文件
    [ -f "$ENV/ip2region.xdb" ] && [ ! -f "$DATA/data/ip2region/ip2region.xdb" ] && \
        cp "$ENV/ip2region.xdb" "$DATA/data/ip2region/ip2region.xdb"
    [ ! -f "$DATA/data/friendb/friends.friendb" ] && touch "$DATA/data/friendb/friends.friendb"

    # 前端软链接 → 补丁仓库编译产物
    rm -rf "$DATA/html/assets" "$DATA/html/index.html" "$DATA/html/images"
    ln -sf "$WEB_DIST/assets"     "$DATA/html/assets"
    ln -sf "$WEB_DIST/index.html" "$DATA/html/index.html"
    [ -d "$WEB_DIST/images" ] && ln -sf "$WEB_DIST/images" "$DATA/html/images"
    log_ok "前端软链接 → $WEB_DIST"
}

# ============================================================
# [8] 启动/停止服务
# ============================================================
do_start() {
    step "启动服务 (端口 $HTTP_PORT)"
    cd "$DATA"

    # 环境变量
    export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
    export SYSTEM_THREAD_LIMIT=1 SYSTEM_MSG_LIMIT=100 SYSTEM_DBLOAD=true
    export HTTP_SERVER1_URL=:$HTTP_PORT HTTP_SERVER1_MODE=debug
    export HTTP_SERVER1_AUTH_PASSWORD=q2-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
    export HTTP_SERVER1_HTML_PATH=./html
    export MONGO1_URL=localhost:27017/${MONGO_DB}?authSource=admin MONGO1_DATABASE=$MONGO_DB
    export MONGO1_USERNAME=$MONGO_USER MONGO1_PASSWORD=$MONGO_PASS
    export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=$INFLUX_ORG INFLUX1_TOKEN=$INFLUX_TOKEN
    export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
    export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
    export DRIVE_MAPPING=server1#client#8.9.80#http://8.130.31.166:8098
    export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

    pkill -f q2-env-patch 2>/dev/null; sleep 1
    nohup "$BINARY" > "$BASE/server.log" 2>&1 &
    sleep 2

    if pgrep -f q2-env-patch > /dev/null; then
        log_ok "服务运行中 PID: $(pgrep -f q2-env-patch)"
    else
        log_err "启动失败，查看: tail -f $BASE/server.log"
    fi
}

do_stop() {
    step "停止服务"
    pkill -f q2-env-patch 2>/dev/null && log_ok "服务已停止" || log_warn "服务未在运行"
}

# ============================================================
# 部署完成信息
# ============================================================
do_show_result() {
    IP=$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    echo ""
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  部署完成!${NC}"
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
    echo -e "  访问: ${CYAN}http://${IP}:${HTTP_PORT}${NC}"
    echo ""
    echo -e "  源码（不可修改）:  $SRC_GO"
    echo -e "                     $SRC_UI"
    echo -e "  补丁仓库:          $ENV  ← $GIT_REPO"
    echo ""
    echo -e "  ${YELLOW}bash server-deploy.sh${NC}            全量部署"
    echo -e "  ${YELLOW}bash server-deploy.sh build${NC}      编译"
    echo -e "  ${YELLOW}bash server-deploy.sh start${NC}      启动"
    echo -e "  ${YELLOW}bash server-deploy.sh update${NC}     更新(git pull+编译+重启)"
    echo -e "  ${YELLOW}bash server-deploy.sh stop${NC}       停止"
    echo ""
}

# ============================================================
# 主入口
# ============================================================
case "${1:-deploy}" in
    deploy)
        echo -e "\n${GREEN}══════════════════════════════════════════════${NC}"
        echo -e "${GREEN}  YMLink-Q2 一键部署（司法鉴定版）${NC}"
        echo -e "${GREEN}══════════════════════════════════════════════${NC}"
        echo -e "  Go 源码:  ${CYAN}$SRC_GO${NC}"
        echo -e "  Vue 源码: ${CYAN}$SRC_UI${NC}"
        echo -e "  补丁仓库: ${CYAN}$GIT_REPO → $ENV${NC}"

        do_check_src
        do_install_deps
        do_clone_repo
        do_build_go
        do_build_frontend
        do_install_db
        do_init_data
        do_start
        do_show_result
        ;;
    build)
        do_check_src
        do_build_go
        do_build_frontend
        ;;
    start)
        do_start
        ;;
    stop)
        do_stop
        ;;
    update)
        step "拉取最新补丁"
        cd "$ENV" && git pull
        do_check_src
        do_build_go
        do_build_frontend
        do_start
        log_ok "更新完成"
        ;;
    *)
        echo "用法: bash $0 {deploy|build|start|update|stop}"
        exit 1
        ;;
esac
