#!/bin/bash
# ============================================================
# YMLink-Q2 补丁构建脚本
# 使用 Go overlay 机制，不修改原始源码，编译时替换补丁文件
#
# 用法（本地Mac）:  ./build-patched.sh
# 用法（服务器）:    ./build-patched.sh --server
# ============================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PATCH_DIR="$SCRIPT_DIR"
OVERLAY_JSON="$PATCH_DIR/overlay.json"

# 判断运行环境
if [ "$1" = "--server" ]; then
    SRC_DIR="/root/q2/ymlink-q2-new-master"
    OUTPUT="/root/q2/ymlink-server"
else
    ENV_DIR="$(dirname "$SCRIPT_DIR")"
    SRC_DIR="$(dirname "$ENV_DIR")/ymlink-q2-new-master"
    OUTPUT="$SRC_DIR/ymlink-server"
fi

echo "=========================================="
echo "  YMLink-Q2 补丁构建"
echo "  源码: $SRC_DIR"
echo "  补丁: $PATCH_DIR"
echo "  输出: $OUTPUT"
echo "=========================================="

# [1/3] 自动生成 overlay.json — 扫描补丁目录中所有 .go 文件
echo "[1/3] 生成 overlay.json ..."

FIRST=true
COUNT=0
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
        COUNT=$((COUNT + 1))
        echo "  + $rel" >&2
    done < <(find "$PATCH_DIR" -name "*.go" -type f | sort)
    echo ''
    echo '  }'
    echo '}'
} > "$OVERLAY_JSON"

echo "  共 $COUNT 个补丁文件"

# [2/3] 编译
echo ""
echo "[2/3] 编译中 (go build -overlay) ..."
cd "$SRC_DIR"
export GOPROXY=https://goproxy.cn,direct
go build -overlay="$OVERLAY_JSON" -o "$OUTPUT" ./apps/server/
echo "  编译成功: $OUTPUT"

# [3/3] 验证
echo ""
echo "[3/3] 验证 ..."
ls -lh "$OUTPUT" | awk '{print "  文件大小:", $5, " 修改时间:", $6, $7, $8}'

echo ""
echo "=========================================="
echo "  构建完成！共 $COUNT 个补丁。"
echo "  原始源码未做任何修改。"
echo "=========================================="
