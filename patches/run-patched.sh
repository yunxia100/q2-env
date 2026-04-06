#!/bin/bash
# ============================================================
# YMLink-Q2 补丁直接运行（不编译，go run + overlay）
# 用法: ./run-patched.sh
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PATCH_DIR="$SCRIPT_DIR"
OVERLAY_JSON="$PATCH_DIR/overlay.json"
ENV_DIR="$(dirname "$SCRIPT_DIR")"
SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"

# 自动生成 overlay.json
FIRST=true
{
    echo '{'
    echo '  "Replace": {'
    while IFS= read -r patch_file; do
        rel="${patch_file#$PATCH_DIR/}"
        original="$SRC_DIR/$rel"
        if [ "$FIRST" = true ]; then
            FIRST=false
        else
            echo ','
        fi
        printf '    "%s": "%s"' "$original" "$patch_file"
    done < <(find "$PATCH_DIR" -name "*.go" -type f | sort)
    echo ''
    echo '  }'
    echo '}'
} > "$OVERLAY_JSON"

cd "$SRC_DIR"
export GOPROXY=https://goproxy.cn,direct

# -------- 环境变量 --------
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
export HTTP_SERVER1_HTML_PATH=./html

export MONGO1_URL='localhost:27017/ymlink_q2?authSource=admin'
export MONGO1_DATABASE=ymlink_q2
export MONGO1_USERNAME=admin
export MONGO1_PASSWORD=admin

export INFLUX1_URL=http://localhost:8086
export INFLUX1_ORG=ymlink
export INFLUX1_TOKEN=ymlink-influx-token

export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
export FRIENDB1_TOTAL=100000

export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb

export DRIVE_MAPPING='server1#androidQQ#8.9.80#http://localhost:8098'
export DRIVE_TIMEOUT=30
export DRIVE_MAX_CONN=50
export MOBILE_PORT=16100

echo "=========================================="
echo "  YMLink-Q2 (patched) go run"
echo "  HTTP:       http://localhost:8080"
echo "  业务日志:   ./log/ymlink-q2.log"
echo "  请求日志:   ./log/access.log"
echo "  错误日志:   ./log/error.log"
echo "  Ctrl+C 停止"
echo "=========================================="

go run -overlay="$OVERLAY_JSON" ./apps/server/
