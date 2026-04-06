#!/bin/bash
# ============================================================
#  YMLink-Q2 部署验证测试脚本
#
#  用法:  bash server-test.sh
#
#  检查项:
#    1. 原始源码目录完整性（司法鉴定：确认未被修改）
#    2. 补丁仓库 + overlay 文件
#    3. 编译产物（二进制 + 前端）
#    4. 数据库服务（MongoDB + InfluxDB）
#    5. 应用服务 + 端口
#    6. HTTP 接口测试
#    7. 前端页面 + 静态资源
# ============================================================

# ---- 配置（与 server-deploy.sh 保持一致）----
SRC_GO="/root/q2/ymlink-q2-new-master"
SRC_UI="/root/q2/ymlink-q2-ui-main"
BASE="/root/q2"
ENV="/root/env"
PATCHES="$ENV/patches"
BINARY="$ENV/q2-env-patch"
WEB_DIST="$ENV/web-env"
DATA="$BASE/server-data"
HTTP_PORT="8080"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
PASS=0; FAIL=0; WARN=0

pass() { echo -e "  ${GREEN}✓${NC} $1"; PASS=$((PASS + 1)); }
fail() { echo -e "  ${RED}✗${NC} $1"; FAIL=$((FAIL + 1)); }
warn() { echo -e "  ${YELLOW}!${NC} $1"; WARN=$((WARN + 1)); }
section() { echo -e "\n${CYAN}[$1]${NC} $2"; }

echo ""
echo -e "${CYAN}══════════════════════════════════════════════${NC}"
echo -e "${CYAN}  YMLink-Q2 部署验证测试${NC}"
echo -e "${CYAN}══════════════════════════════════════════════${NC}"

# ============================================================
# [1] 原始源码完整性
# ============================================================
section "1/7" "原始源码目录（司法鉴定）"

if [ -d "$SRC_GO" ] && [ -f "$SRC_GO/go.mod" ]; then
    GO_FILES=$(find "$SRC_GO" -name "*.go" -type f | wc -l)
    pass "Go 后端源码: $SRC_GO ($GO_FILES 个 .go 文件)"
    # 检查源码是否被 git 追踪且干净
    if [ -d "$SRC_GO/.git" ]; then
        cd "$SRC_GO"
        if git diff --quiet 2>/dev/null; then
            pass "Go 源码 git 状态: 干净（未被修改）"
        else
            fail "Go 源码 git 状态: 有修改！（司法鉴定不通过）"
        fi
    else
        warn "Go 源码无 git 追踪，无法自动验证是否被修改"
    fi
else
    fail "Go 后端源码不存在: $SRC_GO"
fi

if [ -d "$SRC_UI" ] && [ -f "$SRC_UI/package.json" ]; then
    UI_FILES=$(find "$SRC_UI/src" -type f 2>/dev/null | wc -l)
    pass "Vue 前端源码: $SRC_UI ($UI_FILES 个源文件)"
    if [ -d "$SRC_UI/.git" ]; then
        cd "$SRC_UI"
        if git diff --quiet 2>/dev/null; then
            pass "前端源码 git 状态: 干净（未被修改）"
        else
            fail "前端源码 git 状态: 有修改！（司法鉴定不通过）"
        fi
    else
        warn "前端源码无 git 追踪，无法自动验证是否被修改"
    fi
else
    fail "Vue 前端源码不存在: $SRC_UI"
fi

# ============================================================
# [2] 补丁仓库 + overlay
# ============================================================
section "2/7" "补丁仓库"

if [ -d "$ENV/.git" ]; then
    BRANCH=$(cd "$ENV" && git branch --show-current 2>/dev/null)
    COMMIT=$(cd "$ENV" && git log -1 --format="%h %s" 2>/dev/null)
    pass "补丁仓库: $ENV (分支: $BRANCH)"
    pass "最新提交: $COMMIT"
else
    fail "补丁仓库不存在: $ENV"
fi

if [ -d "$PATCHES" ]; then
    GO_PATCHES=$(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | wc -l)
    WEB_PATCHES_COUNT=$(find "$PATCHES/.web" -type f 2>/dev/null | wc -l)
    pass "Go 补丁: $GO_PATCHES 个文件"
    pass "前端补丁: $WEB_PATCHES_COUNT 个文件"
else
    fail "补丁目录不存在: $PATCHES"
fi

if [ -f "$PATCHES/overlay.json" ]; then
    OVERLAY_COUNT=$(grep -c '"' "$PATCHES/overlay.json" 2>/dev/null)
    pass "overlay.json 存在 ($OVERLAY_COUNT 行映射)"
else
    warn "overlay.json 不存在（运行 build 时会自动生成）"
fi

# ============================================================
# [3] 编译产物
# ============================================================
section "3/7" "编译产物"

if [ -f "$BINARY" ] && [ -x "$BINARY" ]; then
    SIZE=$(ls -lh "$BINARY" | awk '{print $5}')
    MTIME=$(stat -c '%y' "$BINARY" 2>/dev/null | cut -d. -f1)
    pass "Go 二进制: $BINARY ($SIZE, $MTIME)"
else
    fail "Go 二进制不存在或不可执行: $BINARY"
fi

if [ -d "$WEB_DIST" ] && [ -f "$WEB_DIST/index.html" ]; then
    ASSET_COUNT=$(find "$WEB_DIST/assets" -type f 2>/dev/null | wc -l)
    pass "前端产物: $WEB_DIST ($ASSET_COUNT 个资源文件)"
else
    fail "前端产物不存在: $WEB_DIST"
fi

# ============================================================
# [4] 数据库服务
# ============================================================
section "4/7" "数据库服务"

# MongoDB
if docker exec q2-mongo mongosh --quiet --eval 'db.runCommand({ping:1})' \
    -u admin -p admin --authenticationDatabase admin >/dev/null 2>&1; then
    COLLECTIONS=$(docker exec q2-mongo mongosh q2_db --quiet -u admin -p admin \
        --authenticationDatabase admin --eval 'db.getCollectionNames().length' 2>/dev/null)
    pass "MongoDB 运行中 (q2_db: $COLLECTIONS 个集合)"
elif systemctl is-active mongod &>/dev/null; then
    COLLECTIONS=$(mongosh q2_db --quiet -u admin -p admin \
        --authenticationDatabase admin --eval 'db.getCollectionNames().length' 2>/dev/null)
    pass "MongoDB 运行中 - apt 安装 (q2_db: $COLLECTIONS 个集合)"
else
    fail "MongoDB 未运行"
fi

# InfluxDB
if curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass; then
    BUCKETS=$(docker exec q2-influx influx bucket list --org q2org --token q2-influx-token 2>/dev/null | grep -c "realtime\|history")
    pass "InfluxDB 运行中 ($BUCKETS 个 bucket)"
else
    fail "InfluxDB 未运行"
fi

# ============================================================
# [5] 应用服务 + 端口
# ============================================================
section "5/7" "应用服务"

if pgrep -f q2-env-patch > /dev/null; then
    PID=$(pgrep -f q2-env-patch)
    MEM=$(ps -p $PID -o rss= 2>/dev/null | awk '{printf "%.0f MB", $1/1024}')
    pass "q2-env-patch 运行中 (PID: $PID, 内存: $MEM)"
else
    fail "q2-env-patch 未运行"
fi

if ss -tlnp | grep -q ":$HTTP_PORT "; then
    pass "端口 $HTTP_PORT 已监听"
else
    fail "端口 $HTTP_PORT 未监听"
fi

# ============================================================
# [6] HTTP 接口测试
# ============================================================
section "6/7" "HTTP 接口测试"

test_url() {
    local url="$1" desc="$2" expect_code="${3:-200}"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$url" 2>/dev/null)
    if [ "$code" = "$expect_code" ]; then
        pass "$desc → $code"
    else
        fail "$desc → $code (期望 $expect_code)"
    fi
}

LOCAL="http://localhost:$HTTP_PORT"

test_url "$LOCAL/"                                         "首页 HTML"
test_url "$LOCAL/api/user/fetch"                          "用户接口 (无token)" "401"
test_url "$LOCAL/api/robot/batch/info?key=test"           "生成器接口"          "400"
test_url "$LOCAL/api/proxy/fetch"                         "代理接口 (无token)" "401"

# 尝试登录获取 token 测试更多接口
TOKEN=$(curl -s -X POST "$LOCAL/api/user/signin" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"a12345677"}' 2>/dev/null | \
    grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
    pass "登录成功，获取到 token"
    test_auth() {
        local url="$1" desc="$2"
        local code
        code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 \
            -H "Authorization: Bearer $TOKEN" "$url" 2>/dev/null)
        if [ "$code" = "200" ]; then
            pass "$desc → $code"
        else
            fail "$desc → $code"
        fi
    }
    test_auth "$LOCAL/api/user/fetch"       "用户列表"
    test_auth "$LOCAL/api/user/environment" "系统环境"
    test_auth "$LOCAL/api/user/status"      "用户状态"
    test_auth "$LOCAL/api/robot/fetch?size=10&index=1"  "机器人列表"
    test_auth "$LOCAL/api/robot/label/fetch" "标签列表"
    test_auth "$LOCAL/api/proxy/fetch"       "代理列表"
    test_auth "$LOCAL/api/task/fetch?size=10&index=1"   "任务列表"
    test_auth "$LOCAL/api/custservice/fetch" "客服列表"
else
    warn "登录失败，跳过认证接口测试（可能无 admin 用户）"
fi

# ============================================================
# [7] 前端静态资源
# ============================================================
section "7/7" "前端静态资源"

# 从 index.html 提取实际的 JS/CSS 文件名
JS_FILE=$(curl -s "$LOCAL/" 2>/dev/null | grep -o 'src="/assets/index-[^"]*\.js"' | head -1 | cut -d'"' -f2)
CSS_FILE=$(curl -s "$LOCAL/" 2>/dev/null | grep -o 'href="/assets/index-[^"]*\.css"' | head -1 | cut -d'"' -f2)

[ -n "$JS_FILE" ]  && test_url "$LOCAL$JS_FILE"  "JS: $JS_FILE"  || warn "未找到 JS 文件引用"
[ -n "$CSS_FILE" ] && test_url "$LOCAL$CSS_FILE" "CSS: $CSS_FILE" || warn "未找到 CSS 文件引用"
test_url "$LOCAL/images/icon/logo.svg" "Logo 图标"

# 软链接检查
if [ -L "$DATA/html/assets" ]; then
    TARGET=$(readlink "$DATA/html/assets")
    pass "html/assets → $TARGET"
else
    fail "html/assets 不是软链接"
fi

# ============================================================
# 汇总
# ============================================================
echo ""
echo -e "${CYAN}══════════════════════════════════════════════${NC}"
TOTAL=$((PASS + FAIL + WARN))
echo -e "  测试完成: $TOTAL 项"
echo -e "  ${GREEN}通过: $PASS${NC}  ${RED}失败: $FAIL${NC}  ${YELLOW}警告: $WARN${NC}"
echo -e "${CYAN}══════════════════════════════════════════════${NC}"

if [ $FAIL -eq 0 ]; then
    echo -e "\n  ${GREEN}★ 全部通过${NC}\n"
    exit 0
else
    echo -e "\n  ${RED}✗ 有 $FAIL 项失败，请检查${NC}\n"
    exit 1
fi
