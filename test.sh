#!/bin/bash
# ============================================================
#  YMLink-Q2 功能测试脚本
#
#  用法:
#    bash test.sh setup       创建标签 + 生成器
#    bash test.sh proxy       检查静态代理
#    bash test.sh login       提交账号并触发登录
#    bash test.sh check       检查机器人列表和状态
#    bash test.sh clean       清理测试数据
#    bash test.sh all         一键执行全流程
#    bash test.sh deploy      部署验证（原有测试）
#
#  支持环境变量覆盖:
#    HOST=http://1.2.3.4:8080 bash test.sh all
# ============================================================

# ************************************************************
# *                    配置区                                  *
# ************************************************************

# 服务地址
HOST="${HOST:-http://localhost:8080}"

# 管理员账号
ADMIN_USER="admin"
ADMIN_PASS="a12345677"

# 生成器配置
BATCH_NAME="test-auto"
BATCH_HARDWARE="iPhone"
BATCH_SOFTWARE="iOSQQ"
BATCH_VERSION="9.1.75"
LABEL_NAME="test-345"

# 默认测试账号（格式: 账号----密码----objid）
# 有 objid 的走复用设备，没有的创建新设备
DEFAULT_ACCOUNTS=(
    "2137274756----hdun7103----XCNFu"
)

# 等待登录的超时秒数
LOGIN_WAIT=30
# 检查间隔
CHECK_INTERVAL=3

# ************************************************************
# *                    颜色和工具函数                           *
# ************************************************************

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
PASS=0; FAIL=0; WARN=0

pass() { echo -e "  ${GREEN}[OK]${NC}  $1"; PASS=$((PASS+1)); }
fail() { echo -e "  ${RED}[FAIL]${NC} $1"; FAIL=$((FAIL+1)); }
warn() { echo -e "  ${YELLOW}[!]${NC}   $1"; WARN=$((WARN+1)); }
info() { echo -e "  ${CYAN}-->  ${NC} $1"; }
section() { echo -e "\n${CYAN}==> $1${NC}"; }
divider() { echo -e "${CYAN}══════════════════════════════════════════════${NC}"; }

# TOKEN 缓存
TOKEN=""

get_token() {
    if [ -n "$TOKEN" ]; then
        return 0
    fi
    local resp
    resp=$(curl -sf -X POST "$HOST/api/user/signin" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" 2>/dev/null)
    TOKEN=$(echo "$resp" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ -z "$TOKEN" ]; then
        fail "登录失败，无法获取 token (resp: $resp)"
        return 1
    fi
    return 0
}

# 带认证的 GET 请求，返回 body
auth_get() {
    curl -sf -H "Authorization: Bearer $TOKEN" "$HOST$1" 2>/dev/null
}

# 带认证的 POST 请求
auth_post() {
    curl -sf -X POST -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "$2" "$HOST$1" 2>/dev/null
}

# 无认证 POST
noauth_post() {
    curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$2" "$HOST$1" 2>/dev/null
}

# 从 JSON 提取字段（轻量，无需 jq 依赖）
json_val() {
    echo "$1" | grep -o "\"$2\":[^,}]*" | head -1 | sed "s/\"$2\"://;s/\"//g"
}

# 从 JSON 数组提取第一个对象的字段
json_arr_first() {
    echo "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -1 | cut -d'"' -f4
}

# ************************************************************
# *                    setup - 创建标签和生成器                 *
# ************************************************************

cmd_setup() {
    section "创建标签和生成器"

    get_token || return 1
    pass "管理员登录成功"

    # --- 检查/创建标签 ---
    info "检查标签: $LABEL_NAME"
    local labels_resp label_id
    labels_resp=$(auth_get "/api/robot/label/fetch")

    label_id=$(echo "$labels_resp" | grep -o "\"id\":\"[^\"]*\",\"name\":\"$LABEL_NAME\"" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

    if [ -z "$label_id" ]; then
        # 也尝试 name 在 id 前面的格式
        label_id=$(echo "$labels_resp" | grep -o "\"name\":\"$LABEL_NAME\"" -A0 -B10 2>/dev/null)
        # 通过 python/node 解析更可靠
        label_id=""
    fi

    if [ -z "$label_id" ]; then
        info "创建标签: $LABEL_NAME"
        local create_resp
        create_resp=$(auth_post "/api/robot/label/create" "{\"name\":\"$LABEL_NAME\"}")
        if echo "$create_resp" | grep -q '"code":200'; then
            pass "标签创建成功"
            # 重新获取标签列表拿 ID
            labels_resp=$(auth_get "/api/robot/label/fetch")
        else
            fail "标签创建失败: $create_resp"
        fi
    else
        pass "标签已存在: $label_id"
    fi

    # 用 python 解析 label_id（更可靠）
    label_id=$(python3 -c "
import json,sys
try:
    data = json.loads('''$labels_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list):
        for item in items:
            if item.get('name') == '$LABEL_NAME':
                print(item.get('id',''))
                break
except: pass
" 2>/dev/null)

    if [ -n "$label_id" ]; then
        pass "标签 ID: $label_id"
    else
        warn "无法解析标签 ID，生成器将不绑定标签"
        label_id=""
    fi

    # --- 检查/创建生成器 ---
    info "检查生成器: $BATCH_NAME"
    local batch_resp batch_key
    batch_resp=$(auth_get "/api/robot/batch/fetch")

    batch_key=$(python3 -c "
import json
try:
    data = json.loads('''$batch_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list):
        for item in items:
            if item.get('name') == '$BATCH_NAME':
                print(item.get('key',{}).get('value',''))
                break
except: pass
" 2>/dev/null)

    if [ -n "$batch_key" ]; then
        pass "生成器已存在，key: $batch_key"
    else
        info "创建生成器: $BATCH_NAME"

        local label_arr="[]"
        [ -n "$label_id" ] && label_arr="[\"$label_id\"]"

        local body="{
            \"name\": \"$BATCH_NAME\",
            \"mode\": \"account\",
            \"label_ids\": $label_arr,
            \"device\": {
                \"hardware\": \"$BATCH_HARDWARE\",
                \"software\": \"$BATCH_SOFTWARE\",
                \"software_version\": \"$BATCH_VERSION\"
            },
            \"config\": {
                \"account\": {
                    \"try_limit\": 5,
                    \"nick_material_name\": \"\"
                }
            }
        }"

        local create_resp
        create_resp=$(auth_post "/api/robot/batch/create" "$body")

        if echo "$create_resp" | grep -q '"code":200'; then
            pass "生成器创建成功"
            # 重新获取 key
            batch_resp=$(auth_get "/api/robot/batch/fetch")
            batch_key=$(python3 -c "
import json
try:
    data = json.loads('''$batch_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list):
        for item in items:
            if item.get('name') == '$BATCH_NAME':
                print(item.get('key',{}).get('value',''))
                break
except: pass
" 2>/dev/null)
            if [ -n "$batch_key" ]; then
                pass "生成器 key: $batch_key"
            else
                fail "无法获取生成器 key"
            fi
        else
            fail "生成器创建失败: $create_resp"
        fi
    fi

    # 保存 key 到临时文件供后续步骤使用
    echo "$batch_key" > /tmp/q2_test_batch_key
    info "key 已保存到 /tmp/q2_test_batch_key"

    # --- 验证生成器可访问 ---
    if [ -n "$batch_key" ]; then
        local info_resp
        info_resp=$(curl -sf "$HOST/api/robot/batch/info?key=$batch_key&no_robots=true" 2>/dev/null)
        if echo "$info_resp" | grep -q '"code":200'; then
            local batch_mode
            batch_mode=$(python3 -c "
import json
try:
    data = json.loads('''$info_resp''')
    rb = data.get('data',{}).get('robot_batch',{})
    print(f\"{rb.get('name','')} | 模式: {rb.get('mode','')} | 设备: {rb.get('device',{}).get('hardware','')}/{rb.get('device',{}).get('software','')}/{rb.get('device',{}).get('software_version','')}\")
except: pass
" 2>/dev/null)
            pass "生成器信息: $batch_mode"
            info "生成器页面: $HOST/robot-batch?key=$batch_key"
        else
            fail "无法访问生成器: $info_resp"
        fi
    fi
}

# ************************************************************
# *                    proxy - 检查静态代理                    *
# ************************************************************

cmd_proxy() {
    section "检查静态代理"

    get_token || return 1

    local proxy_resp proxy_count proxy_detail
    proxy_resp=$(auth_get "/api/proxy/fetch")

    proxy_detail=$(python3 -c "
import json
try:
    data = json.loads('''$proxy_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if not isinstance(items, list):
        items = []
    total = len(items)
    active = sum(1 for p in items if not p.get('disabled', False))
    print(f'total={total}')
    print(f'active={active}')
    for p in items[:10]:
        c = p.get('config', {})
        s = p.get('status', {})
        d = '禁用' if p.get('disabled') else '可用'
        robots = s.get('robot_total', 0)
        province = c.get('province', '')
        prot = c.get('protocol', '')
        print(f'  [{d}] {prot}://{c.get(\"ip\",\"?\")}:{c.get(\"port\",\"?\")} {province} (已分配: {robots})')
except Exception as e:
    print(f'error={e}')
" 2>/dev/null)

    local total active
    total=$(echo "$proxy_detail" | grep '^total=' | cut -d= -f2)
    active=$(echo "$proxy_detail" | grep '^active=' | cut -d= -f2)

    if [ "${total:-0}" -gt 0 ]; then
        pass "代理总数: $total, 可用: $active"
        echo "$proxy_detail" | grep '^\s*\[' | while IFS= read -r line; do
            info "$line"
        done
    else
        warn "没有代理！账号提交后将无代理可分配"
        warn "请先在管理后台添加静态代理"
    fi
}

# ************************************************************
# *                    login - 提交账号并登录                  *
# ************************************************************

cmd_login() {
    section "提交账号并触发登录"

    # 读取 batch key
    local batch_key
    if [ -f /tmp/q2_test_batch_key ]; then
        batch_key=$(cat /tmp/q2_test_batch_key)
    fi

    if [ -z "$batch_key" ]; then
        # 尝试从服务器获取
        get_token || return 1
        local batch_resp
        batch_resp=$(auth_get "/api/robot/batch/fetch")
        batch_key=$(python3 -c "
import json
try:
    data = json.loads('''$batch_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list):
        for item in items:
            if item.get('name') == '$BATCH_NAME':
                print(item.get('key',{}).get('value',''))
                break
        else:
            if items:
                print(items[0].get('key',{}).get('value',''))
except: pass
" 2>/dev/null)
    fi

    if [ -z "$batch_key" ]; then
        fail "找不到生成器 key，请先运行: bash test.sh setup"
        return 1
    fi

    pass "使用生成器 key: $batch_key"

    # 用命令行参数的账号或默认账号
    local accounts=()
    if [ ${#CMD_ACCOUNTS[@]} -gt 0 ]; then
        accounts=("${CMD_ACCOUNTS[@]}")
    else
        accounts=("${DEFAULT_ACCOUNTS[@]}")
    fi

    info "提交 ${#accounts[@]} 个账号..."

    # 构建 JSON lines 数组
    local lines_json="["
    local first=true
    for acc in "${accounts[@]}"; do
        if [ "$first" = true ]; then
            first=false
        else
            lines_json+=","
        fi
        lines_json+="\"$acc\""
    done
    lines_json+="]"

    local submit_resp
    submit_resp=$(noauth_post "/api/robot/batch/account_submit" \
        "{\"key\":\"$batch_key\",\"lines\":$lines_json}")

    if echo "$submit_resp" | grep -q '"code":200'; then
        pass "账号提交成功"
    else
        fail "账号提交失败: $submit_resp"
        return 1
    fi

    # 解析每个账号的提交结果
    python3 -c "
import json
try:
    data = json.loads('''$submit_resp''')
    results = data.get('data', [])
    for r in results:
        uid = r.get('uid', '?')
        ok = r.get('success', False)
        msg = r.get('msg', '')
        objid = r.get('objid', '')
        reused = r.get('reused', False)
        tag = '复用' if reused else '新建'
        status = 'OK' if ok else 'FAIL'
        print(f'  [{status}] QQ {uid} | objid={objid} | {tag} | {msg}')
except: pass
" 2>/dev/null

    # --- 触发登录 ---
    info "触发登录... (GET /api/robot/login?key=$batch_key)"
    local login_resp
    login_resp=$(curl -sf "$HOST/api/robot/login?key=$batch_key" 2>/dev/null)
    info "登录触发响应: $(echo "$login_resp" | head -c 300)"

    # --- 等待登录完成 ---
    section "等待登录 (最多 ${LOGIN_WAIT}s)"
    local elapsed=0
    while [ $elapsed -lt $LOGIN_WAIT ]; do
        sleep $CHECK_INTERVAL
        elapsed=$((elapsed + CHECK_INTERVAL))

        # 检查所有提交的账号状态
        local all_done=true
        for acc in "${accounts[@]}"; do
            local uid
            uid=$(echo "$acc" | sed 's/----.*//')

            local status_code
            status_code=$(python3 -c "
import json, urllib.request
try:
    req = urllib.request.Request('$HOST/api/robot/batch/info?key=$batch_key')
    resp = urllib.request.urlopen(req, timeout=5)
    data = json.loads(resp.read())
    robots = data.get('data',{}).get('robots',[])
    for r in robots:
        if r.get('submit',{}).get('uid') == $uid:
            login = r.get('status',{}).get('login')
            if login:
                print(login.get('code', -1))
            else:
                print('nil')
            break
    else:
        print('not_found')
except Exception as e:
    print(f'err:{e}')
" 2>/dev/null)

            case "$status_code" in
                0)
                    info "${elapsed}s  QQ $uid: 登录成功 ✓"
                    ;;
                nil|"")
                    info "${elapsed}s  QQ $uid: 等待登录中..."
                    all_done=false
                    ;;
                not_found)
                    info "${elapsed}s  QQ $uid: 未找到"
                    all_done=false
                    ;;
                1)  info "${elapsed}s  QQ $uid: 密码错误"; ;;
                2)  info "${elapsed}s  QQ $uid: 需要滑块验证"; ;;
                9|10) info "${elapsed}s  QQ $uid: 连接中..."; all_done=false ;;
                15) info "${elapsed}s  QQ $uid: 身份验证失效"; ;;
                16) info "${elapsed}s  QQ $uid: 登录失效"; ;;
                40) info "${elapsed}s  QQ $uid: 账号冻结"; ;;
                154) info "${elapsed}s  QQ $uid: 连接超时"; all_done=false ;;
                160) info "${elapsed}s  QQ $uid: 需要短信验证"; ;;
                239) info "${elapsed}s  QQ $uid: 需要辅助验证"; ;;
                *)  info "${elapsed}s  QQ $uid: 状态码=$status_code"; ;;
            esac
        done

        if [ "$all_done" = true ]; then
            break
        fi
    done

    echo ""
    # 最终状态汇总
    for acc in "${accounts[@]}"; do
        local uid
        uid=$(echo "$acc" | sed 's/----.*//')
        local final_code
        final_code=$(python3 -c "
import json, urllib.request
try:
    req = urllib.request.Request('$HOST/api/robot/batch/info?key=$batch_key')
    resp = urllib.request.urlopen(req, timeout=5)
    data = json.loads(resp.read())
    robots = data.get('data',{}).get('robots',[])
    for r in robots:
        if r.get('submit',{}).get('uid') == $uid:
            login = r.get('status',{}).get('login')
            if login:
                print(login.get('code', -1))
            else:
                print('nil')
            break
except: print('err')
" 2>/dev/null)
        if [ "$final_code" = "0" ]; then
            pass "QQ $uid: 登录成功"
        elif [ "$final_code" = "nil" ] || [ -z "$final_code" ]; then
            warn "QQ $uid: 未完成登录 (status=nil，系统可能仍在处理)"
        else
            fail "QQ $uid: 登录未成功 (code=$final_code)"
        fi
    done
}

# ************************************************************
# *                    check - 检查机器人列表                  *
# ************************************************************

cmd_check() {
    section "检查机器人列表"

    get_token || return 1

    # 获取机器人列表
    local robot_resp
    robot_resp=$(auth_get "/api/robot/fetch?size=50&index=1")

    python3 -c "
import json
try:
    data = json.loads('''$robot_resp''')
    items = data.get('data', data)
    if isinstance(items, dict):
        items = items.get('list', items.get('data', []))
    if not isinstance(items, list):
        print('无法解析机器人列表')
    else:
        print(f'  机器人总数: {len(items)}')
        print()
        login_codes = {
            0: '已登录', 1: '密码错误', 2: '需滑块', 9: '连接中',
            15: '失效', 16: '登录失效', 20: '过期', 40: '冻结',
            154: '超时', 160: '需短信', 239: '需辅助'
        }
        for r in items:
            uid = r.get('submit',{}).get('uid', '?')
            objid = r.get('kernel',{}).get('objid', '?')
            login = r.get('status',{}).get('login')
            login_str = '未登录'
            if login:
                code = login.get('code', -1)
                login_str = login_codes.get(code, f'code={code}')
            stop = r.get('stop', False)
            deleted = r.get('deleted', False)
            nick = r.get('status',{}).get('profile',{}).get('nick','') if r.get('status',{}).get('profile') else ''
            offline = r.get('cache',{}).get('offline','')

            flags = []
            if deleted: flags.append('已删除')
            if stop: flags.append('已停止')
            if offline: flags.append(f'离线:{offline[:20]}')
            flag_str = ' '.join(flags) if flags else ''

            nick_str = f' ({nick})' if nick else ''
            print(f'  QQ {uid}{nick_str} | objid={objid} | {login_str} {flag_str}')
except Exception as e:
    print(f'  解析失败: {e}')
" 2>/dev/null

    # 检查代理状态
    echo ""
    local proxy_resp
    proxy_resp=$(auth_get "/api/proxy/fetch")
    python3 -c "
import json
try:
    data = json.loads('''$proxy_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list) and len(items) > 0:
        active = sum(1 for p in items if not p.get('disabled', False))
        print(f'  代理: {len(items)} 个 (可用 {active})')
    else:
        print('  代理: 无')
except: print('  代理: 解析失败')
" 2>/dev/null

    # 驱动连通性
    echo ""
    local drive_url="http://8.130.31.166:8098"
    local drive_resp
    drive_resp=$(curl -sf --connect-timeout 3 "$drive_url/" 2>/dev/null)
    if [ $? -eq 0 ]; then
        pass "底层驱动连通: $drive_url"
    else
        fail "底层驱动不可达: $drive_url"
    fi
}

# ************************************************************
# *                    clean - 清理测试数据                    *
# ************************************************************

cmd_clean() {
    section "清理测试数据"

    get_token || return 1

    # 查找测试生成器
    local batch_resp batch_id batch_key
    batch_resp=$(auth_get "/api/robot/batch/fetch")

    read -r batch_id batch_key <<< $(python3 -c "
import json
try:
    data = json.loads('''$batch_resp''')
    items = data.get('data', data) if isinstance(data, dict) else data
    if isinstance(items, list):
        for item in items:
            if item.get('name') == '$BATCH_NAME':
                print(item.get('id',''), item.get('key',{}).get('value',''))
                break
except: pass
" 2>/dev/null)

    if [ -n "$batch_id" ]; then
        info "找到测试生成器: $BATCH_NAME (id=$batch_id, key=$batch_key)"

        # 删除生成器下所有机器人
        local del_robots_resp
        del_robots_resp=$(curl -sf "$HOST/api/robot/delete_by_batch?key=$batch_key" 2>/dev/null)
        info "删除机器人: $del_robots_resp"

        # 删除生成器本身
        local del_batch_resp
        del_batch_resp=$(curl -sf -X DELETE -H "Authorization: Bearer $TOKEN" \
            "$HOST/api/robot/batch/delete?id=$batch_id" 2>/dev/null)
        if echo "$del_batch_resp" | grep -q '"code":200'; then
            pass "测试生成器已删除"
        else
            warn "删除生成器: $del_batch_resp"
        fi
    else
        info "未找到测试生成器 $BATCH_NAME，无需清理"
    fi

    # 清理临时文件
    rm -f /tmp/q2_test_batch_key
    pass "临时文件已清理"
}

# ************************************************************
# *                    deploy - 部署验证                       *
# ************************************************************

cmd_deploy() {
    local SRC_GO="/root/q2/ymlink-q2-new-master"
    local SRC_UI="/root/q2/ymlink-q2-ui-main"
    local BASE="/root/q2"
    local ENV="/root/env"
    local PATCHES="$ENV/patches"
    local BINARY="$ENV/q2-env-patch"
    local WEB_DIST="$ENV/web-env"
    local DATA="$BASE/server-data"
    local HTTP_PORT="8080"

    # [1] 原始源码完整性
    section "原始源码目录（司法鉴定）"
    if [ -d "$SRC_GO" ] && [ -f "$SRC_GO/go.mod" ]; then
        local GO_FILES
        GO_FILES=$(find "$SRC_GO" -name "*.go" -type f | wc -l)
        pass "Go 后端源码: $SRC_GO ($GO_FILES 个 .go 文件)"
        if [ -d "$SRC_GO/.git" ]; then
            cd "$SRC_GO"
            if git diff --quiet 2>/dev/null; then
                pass "Go 源码 git 状态: 干净（未被修改）"
            else
                fail "Go 源码 git 状态: 有修改！（司法鉴定不通过）"
            fi
        fi
    else
        fail "Go 后端源码不存在: $SRC_GO"
    fi

    if [ -d "$SRC_UI" ] && [ -f "$SRC_UI/package.json" ]; then
        pass "Vue 前端源码存在: $SRC_UI"
    else
        fail "Vue 前端源码不存在: $SRC_UI"
    fi

    # [2] 补丁仓库
    section "补丁仓库"
    if [ -d "$ENV/.git" ]; then
        local COMMIT
        COMMIT=$(cd "$ENV" && git log -1 --format="%h %s" 2>/dev/null)
        pass "补丁仓库: $COMMIT"
    else
        fail "补丁仓库不存在: $ENV"
    fi

    if [ -d "$PATCHES" ]; then
        local GO_PATCHES
        GO_PATCHES=$(find "$PATCHES" -name "*.go" -type f -not -path "*/.web/*" | wc -l)
        pass "Go 补丁: $GO_PATCHES 个文件"
    fi

    # [3] 编译产物
    section "编译产物"
    if [ -f "$BINARY" ] && [ -x "$BINARY" ]; then
        local SIZE
        SIZE=$(ls -lh "$BINARY" | awk '{print $5}')
        pass "Go 二进制: $SIZE"
    else
        fail "Go 二进制不存在: $BINARY"
    fi

    if [ -d "$WEB_DIST" ] && [ -f "$WEB_DIST/index.html" ]; then
        pass "前端产物存在"
    else
        fail "前端产物不存在"
    fi

    # [4] 数据库
    section "数据库服务"
    if systemctl is-active mongod &>/dev/null; then
        pass "MongoDB 运行中 (apt)"
    elif docker exec q2-mongo mongosh --quiet --eval 'db.runCommand({ping:1})' \
        -u admin -p admin --authenticationDatabase admin >/dev/null 2>&1; then
        pass "MongoDB 运行中 (docker)"
    else
        fail "MongoDB 未运行"
    fi

    if curl -sf http://localhost:8086/health 2>/dev/null | grep -q pass; then
        pass "InfluxDB 运行中"
    else
        fail "InfluxDB 未运行"
    fi

    # [5] 应用服务
    section "应用服务"
    if pgrep -f q2-env-patch > /dev/null; then
        local PID MEM
        PID=$(pgrep -f q2-env-patch)
        MEM=$(ps -p $PID -o rss= 2>/dev/null | awk '{printf "%.0f MB", $1/1024}')
        pass "q2-env-patch PID=$PID 内存=$MEM"
    else
        fail "q2-env-patch 未运行"
    fi

    if ss -tlnp | grep -q ":$HTTP_PORT "; then
        pass "端口 $HTTP_PORT 已监听"
    else
        fail "端口 $HTTP_PORT 未监听"
    fi

    # [6] HTTP 接口
    section "HTTP 接口"
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$HOST/" 2>/dev/null)
    [ "$code" = "200" ] && pass "首页 → $code" || fail "首页 → $code"

    code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$HOST/api/user/fetch" 2>/dev/null)
    [ "$code" = "200" ] && pass "用户接口 → $code" || fail "用户接口 → $code"

    code=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 "$HOST/api/robot/batch/info?key=test" 2>/dev/null)
    [ "$code" = "200" ] && pass "生成器接口 → $code" || fail "生成器接口 → $code"
}

# ************************************************************
# *                    all - 全流程                            *
# ************************************************************

cmd_all() {
    echo ""
    divider
    echo -e "  ${BOLD}YMLink-Q2 全流程测试${NC}"
    divider

    cmd_deploy
    echo ""
    cmd_setup
    cmd_proxy
    cmd_login
    cmd_check

    # 汇总
    echo ""
    divider
    local TOTAL=$((PASS + FAIL + WARN))
    echo -e "  测试完成: $TOTAL 项"
    echo -e "  ${GREEN}通过: $PASS${NC}  ${RED}失败: $FAIL${NC}  ${YELLOW}警告: $WARN${NC}"
    divider

    if [ $FAIL -eq 0 ]; then
        echo -e "\n  ${GREEN}★ 全部通过${NC}\n"
    else
        echo -e "\n  ${RED}✗ 有 $FAIL 项失败${NC}\n"
    fi
}

# ************************************************************
# *                    入口                                    *
# ************************************************************

usage() {
    echo "用法: bash test.sh <command> [accounts...]"
    echo ""
    echo "命令:"
    echo "  setup        创建标签 + 生成器"
    echo "  proxy        检查静态代理"
    echo "  login        提交账号并触发登录"
    echo "  check        检查机器人列表和状态"
    echo "  clean        清理测试数据"
    echo "  deploy       部署验证"
    echo "  all          一键全流程 (deploy→setup→proxy→login→check)"
    echo ""
    echo "示例:"
    echo "  bash test.sh all"
    echo "  bash test.sh login"
    echo "  bash test.sh login 12345----pwd123"
    echo "  bash test.sh login 12345----pwd123----XCNFu"
    echo "  HOST=http://8.147.71.175:8080 bash test.sh all"
}

CMD="$1"
shift

# 收集命令行传入的额外账号
CMD_ACCOUNTS=()
while [ $# -gt 0 ]; do
    CMD_ACCOUNTS+=("$1")
    shift
done

case "$CMD" in
    setup)  cmd_setup ;;
    proxy)  cmd_proxy ;;
    login)  cmd_login ;;
    check)  cmd_check ;;
    clean)  cmd_clean ;;
    deploy) cmd_deploy ;;
    all)    cmd_all ;;
    -h|--help|help) usage ;;
    *)
        echo -e "${RED}未知命令: $CMD${NC}"
        usage
        exit 1
        ;;
esac
