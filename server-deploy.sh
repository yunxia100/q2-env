#!/bin/bash
# ============================================================
#  YMLink-Q2 一键部署脚本（司法鉴定版）
#
#  用法:
#    全量部署:  bash server-deploy.sh
#    仅编译:    bash server-deploy.sh build
#    仅启动:    bash server-deploy.sh start
#    仅更新:    bash server-deploy.sh update   (git pull + 编译 + 重启)
#    停止服务:  bash server-deploy.sh stop
#
#  原始代码不可修改，补丁通过 Go overlay 机制注入编译
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

# ---- 补丁仓库（git 拉取到 /root/env/）----
GIT_REPO="https://github.com/yunxia100/q2-env.git"
ENV="/root/env"                                 # 补丁仓库目录

# ************************************************************
# *                    以下无需修改                            *
# ************************************************************

PATCHES="$ENV/patches"
BINARY="$ENV/q2-env-patch"
WEB_DIST="$ENV/web-env"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "  ${GREEN}[OK]${NC}  $1"; }
log_warn()  { echo -e "  ${YELLOW}[!]${NC}   $1"; }
log_error() { echo -e "  ${RED}[ERR]${NC} $1"; exit 1; }
log_step()  { echo -e "\n${CYAN}==> $1${NC}"; }

# ============================================================
# 功能函数
# ============================================================

do_check_src() {
    log_step "检查原始源码目录"
    [ ! -d "$SRC_GO" ]            && log_error "Go 源码不存在: $SRC_GO"
    [ ! -f "$SRC_GO/go.mod" ]     && log_error "Go 源码无效（缺少 go.mod）: $SRC_GO"
    log_info "Go 后端源码: $SRC_GO"
    [ ! -d "$SRC_UI" ]              && log_error "前端源码不存在: $SRC_UI"
    [ ! -f "$SRC_UI/package.json" ] && log_error "前端源码无效（缺少 package.json）: $SRC_UI"
    log_info "Vue 前端源码: $SRC_UI"
}

do_install_deps() {
    log_step "安装依赖"
    export DEBIAN_FRONTEND=noninteractive

    # Git + rsync
    command -v git &>/dev/null  && log_info "Git 已安装" || { apt-get update -qq && apt-get install -y -qq git curl jq rsync wget; log_info "Git 安装完成"; }
    command -v rsync &>/dev/null || apt-get install -y -qq rsync 2>/dev/null

    # Docker
    if command -v docker &>/dev/null; then
        log_info "Docker 已安装"
    else
        apt-get install -y -qq docker.io 2>/dev/null || (curl -fsSL https://get.docker.com | sh)
        systemctl enable docker && systemctl start docker
        log_info "Docker 安装完成"
    fi

    # docker-compose
    if command -v docker-compose &>/dev/null || docker compose version &>/dev/null 2>&1; then
        log_info "Docker Compose 就绪"
    else
        apt-get install -y -qq docker-compose 2>/dev/null || apt-get install -y -qq docker-compose-plugin 2>/dev/null || true
        command -v docker-compose &>/dev/null || docker compose version &>/dev/null || log_error "Docker Compose 安装失败"
        log_info "Docker Compose 安装完成"
    fi

    # Go
    export PATH=$PATH:/usr/local/go/bin
    if command -v go &>/dev/null; then
        log_info "Go 已安装: $(go version | awk '{print $3}')"
    else
        GO_VERSION="1.21.13"
        wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz 2>/dev/null || \
        curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz
        rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz && rm -f /tmp/go.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
        go version &>/dev/null || log_error "Go 安装失败"
        log_info "Go 安装完成: $(go version | awk '{print $3}')"
    fi
    export GOPATH=/root/go GOPROXY=https://goproxy.cn,direct

    # Node.js
    if command -v node &>/dev/null; then
        log_info "Node.js 已安装: $(node -v)"
    else
        curl -fsSL https://deb.nodesource.com/setup_18.x | bash - 2>/dev/null
        apt-get install -y -qq nodejs 2>/dev/null
        node -v &>/dev/null || log_error "Node.js 安装失败"
        log_info "Node.js 安装完成: $(node -v)"
    fi
}

do_clone_repo() {
    log_step "拉取补丁仓库 → $ENV"
    if [ -d "$ENV/.git" ]; then
        cd "$ENV" && git pull
        log_info "补丁仓库已更新: $ENV"
    else
        rm -rf "$ENV"
        git clone "$GIT_REPO" "$ENV"
        log_info "补丁仓库克隆完成: $ENV"
    fi
    [ ! -d "$PATCHES" ] && log_error "补丁目录不存在: $PATCHES"
}

do_build_go() {
    log_step "Go overlay 编译（原始源码不修改）"
    export PATH=$PATH:/usr/local/go/bin
    export GOPATH=/root/go GOPROXY=https://goproxy.cn,direct

    OVERLAY_JSON="$PATCHES/overlay.json"
    FIRST=true; COUNT=0
    {
        echo '{'
        echo '  "Replace": {'
        while IFS= read -r pf; do
            rel="${pf#$PATCHES/}"; orig="$SRC_GO/$rel"
            [ "$FIRST" = true ] && FIRST=false || echo ','
            printf '    "%s": "%s"' "$orig" "$pf"
            COUNT=$((COUNT + 1))
        done < <(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | sort)
        echo ''; echo '  }'; echo '}'
    } > "$OVERLAY_JSON"
    log_info "overlay.json: $COUNT 个 Go 补丁"

    pkill -f q2-env-patch 2>/dev/null || true; sleep 1
    cd "$SRC_GO"
    go build -overlay="$OVERLAY_JSON" -o "$BINARY" ./apps/server/
    chmod +x "$BINARY"
    log_info "编译成功: $BINARY ($(ls -lh "$BINARY" | awk '{print $5}'))"
}

do_build_frontend() {
    log_step "前端编译（补丁覆盖 → vite build）"
    WEB_PATCHES="$PATCHES/.web"
    if [ -d "$WEB_PATCHES" ]; then
        UI_WORK="$BASE/.ui-build-temp"
        rm -rf "$UI_WORK"
        cp -a "$SRC_UI" "$UI_WORK"
        rsync -a --exclude='*.md' "$WEB_PATCHES/" "$UI_WORK/"
        log_info "前端补丁: $(find "$WEB_PATCHES" -type f | wc -l) 个文件"
        cd "$UI_WORK"
        npm install --prefer-offline 2>/dev/null || npm install
        npx vite build --outDir "$WEB_DIST"
        rm -rf "$UI_WORK"
        log_info "vite build 完成（原始前端源码未修改）"
    else
        log_warn "无前端补丁，使用仓库已有 web-env/"
    fi
}

do_start_db() {
    log_step "启动数据库"

    # Docker 镜像加速
    if [ ! -f /etc/docker/daemon.json ] || ! grep -q "registry-mirrors" /etc/docker/daemon.json 2>/dev/null; then
        echo '{"registry-mirrors":["https://dockerpull.org","https://docker.rainbond.cc","https://docker.1ms.run"]}' > /etc/docker/daemon.json
        systemctl restart docker && sleep 3
    fi

    cd "$ENV"
    COMPOSE_CMD="docker-compose"; command -v docker-compose &>/dev/null || COMPOSE_CMD="docker compose"
    $COMPOSE_CMD up -d mongodb influxdb

    echo -n "  等待 MongoDB"
    for i in $(seq 1 60); do
        docker exec q2-mongo mongosh --quiet --eval 'db.runCommand({ping:1})' \
            -u admin -p admin --authenticationDatabase admin >/dev/null 2>&1 && { echo ""; log_info "MongoDB 就绪"; break; }
        echo -n "."; sleep 2
        [ $i -eq 60 ] && { echo ""; log_error "MongoDB 超时"; }
    done

    echo -n "  等待 InfluxDB"
    for i in $(seq 1 40); do
        curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass && { echo ""; log_info "InfluxDB 就绪"; break; }
        echo -n "."; sleep 2
        [ $i -eq 40 ] && { echo ""; log_error "InfluxDB 超时"; }
    done

    docker exec q2-influx influx bucket create \
        --name history --org q2org --retention 8760h \
        --token q2-influx-token 2>/dev/null || true
    log_info "InfluxDB buckets 就绪"
}

do_init_data() {
    log_step "初始化运行时目录"
    mkdir -p "$DATA" && cd "$DATA"
    for d in data/friendb data/ip2region log html \
        file/task file/login file/material file/materialdb file/message \
        file/usedb file/qzonedb file/realinfodb file/android_pack \
        file/ios_pack file/ini_pack; do
        mkdir -p "$d"
    done
    [ -f "$ENV/ip2region.xdb" ] && [ ! -f "$DATA/data/ip2region/ip2region.xdb" ] && \
        cp "$ENV/ip2region.xdb" "$DATA/data/ip2region/ip2region.xdb"
    [ ! -f "$DATA/data/friendb/friends.friendb" ] && touch "$DATA/data/friendb/friends.friendb"

    rm -rf "$DATA/html/assets" "$DATA/html/index.html" "$DATA/html/images"
    ln -sf "$WEB_DIST/assets"     "$DATA/html/assets"
    ln -sf "$WEB_DIST/index.html" "$DATA/html/index.html"
    [ -d "$WEB_DIST/images" ] && ln -sf "$WEB_DIST/images" "$DATA/html/images"
    log_info "前端软链接 → $WEB_DIST"
}

do_start() {
    log_step "启动服务 (端口 $HTTP_PORT)"
    cd "$DATA"

    export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
    export SYSTEM_THREAD_LIMIT=1 SYSTEM_MSG_LIMIT=100 SYSTEM_DBLOAD=true
    export HTTP_SERVER1_URL=:$HTTP_PORT HTTP_SERVER1_MODE=debug
    export HTTP_SERVER1_AUTH_PASSWORD=q2-jwt-secret HTTP_SERVER1_AUTH_TIMEOUT=604800
    export HTTP_SERVER1_HTML_PATH=./html
    export MONGO1_URL=localhost:27017/q2_db?authSource=admin MONGO1_DATABASE=q2_db
    export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
    export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=q2org INFLUX1_TOKEN=q2-influx-token
    export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
    export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
    export DRIVE_MAPPING=server1#client#8.9.80#http://8.130.31.166:8098
    export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

    pkill -f q2-env-patch 2>/dev/null || true; sleep 1
    nohup "$BINARY" > "$BASE/server.log" 2>&1 &
    sleep 2

    if pgrep -f q2-env-patch > /dev/null; then
        log_info "服务运行中 PID: $(pgrep -f q2-env-patch)"
    else
        log_error "启动失败，查看: tail -f $BASE/server.log"
    fi
}

do_stop() {
    log_step "停止服务"
    pkill -f q2-env-patch 2>/dev/null && log_info "服务已停止" || log_warn "服务未在运行"
}

do_show_result() {
    PUBLIC_IP=$(curl -s --connect-timeout 3 ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')
    echo ""
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  部署完成!${NC}"
    echo -e "${GREEN}══════════════════════════════════════════════${NC}"
    echo ""
    echo -e "  访问:     ${CYAN}http://${PUBLIC_IP}:${HTTP_PORT}${NC}"
    echo ""
    echo -e "  ${CYAN}源码目录（不可修改）:${NC}"
    echo -e "    Go 后端:  $SRC_GO"
    echo -e "    Vue 前端: $SRC_UI"
    echo ""
    echo -e "  ${CYAN}补丁仓库 (git clone → $ENV):${NC}"
    echo -e "    $GIT_REPO"
    echo ""
    echo -e "  ${YELLOW}常用操作（都用这一个脚本）:${NC}"
    echo -e "    bash server-deploy.sh            全量部署"
    echo -e "    bash server-deploy.sh build       仅编译"
    echo -e "    bash server-deploy.sh start       仅启动"
    echo -e "    bash server-deploy.sh update      git pull + 编译 + 重启"
    echo -e "    bash server-deploy.sh stop        停止"
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
        echo -e "  前端源码: ${CYAN}$SRC_UI${NC}"
        echo -e "  补丁仓库: ${CYAN}$GIT_REPO${NC}"
        echo -e "  端口:     ${CYAN}$HTTP_PORT${NC}"

        do_check_src
        do_install_deps
        do_clone_repo
        do_build_go
        do_build_frontend
        do_start_db
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
        echo -e "${CYAN}=== 拉取最新补丁 ===${NC}"
        cd "$ENV" && git pull
        do_check_src
        do_build_go
        do_build_frontend
        do_start
        log_info "更新完成"
        ;;

    *)
        echo "用法: bash $0 {deploy|build|start|update|stop}"
        exit 1
        ;;
esac
