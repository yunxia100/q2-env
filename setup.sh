#!/bin/bash
# ============================================================
# YMLink-Q2 环境初始化脚本
# 用法: chmod +x setup.sh && ./setup.sh
# ============================================================

set -e

echo "=========================================="
echo "  YMLink-Q2 环境初始化"
echo "=========================================="

# ---------- 1. 创建运行时目录 ----------
echo ""
echo "[1/4] 创建运行时目录..."

dirs=(
    "data/friendb"
    "data/ip2region"
    "file/task"
    "file/material"
    "file/message"
    "file/usedb"
    "file/qzonedb"
    "file/materialdb"
    "file/realinfodb"
    "file/android_pack"
    "file/ios_pack"
    "file/ini_pack"
    "file/login"
)

for dir in "${dirs[@]}"; do
    mkdir -p "$dir"
    echo "  ✓ $dir"
done

# ---------- 2. 下载 ip2region.xdb ----------
echo ""
echo "[2/4] 检查 ip2region.xdb..."

IP2REGION_PATH="data/ip2region/ip2region.xdb"
if [ -f "$IP2REGION_PATH" ]; then
    echo "  ✓ ip2region.xdb 已存在"
else
    echo "  ⬇ 正在下载 ip2region.xdb ..."
    IP2REGION_URL="https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region.xdb"
    if command -v curl &> /dev/null; then
        curl -fSL -o "$IP2REGION_PATH" "$IP2REGION_URL" && echo "  ✓ 下载完成" || echo "  ✗ 下载失败，请手动下载放入 $IP2REGION_PATH"
    elif command -v wget &> /dev/null; then
        wget -q -O "$IP2REGION_PATH" "$IP2REGION_URL" && echo "  ✓ 下载完成" || echo "  ✗ 下载失败，请手动下载放入 $IP2REGION_PATH"
    else
        echo "  ✗ 未找到 curl/wget，请手动下载:"
        echo "    地址: $IP2REGION_URL"
        echo "    放入: $IP2REGION_PATH"
    fi
fi

# ---------- 3. 生成 .env 文件 ----------
echo ""
echo "[3/4] 检查 .env 配置文件..."

if [ -f ".env" ]; then
    echo "  ✓ .env 已存在，跳过"
else
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "  ✓ 已从 .env.example 生成 .env"
        echo "  ⚠ 请编辑 .env 文件填入实际配置值！"
    else
        echo "  ✗ 未找到 .env.example 模板"
    fi
fi

# ---------- 4. 检查 Linux 内核参数 ----------
echo ""
echo "[4/4] 检查系统参数..."

if [ "$(uname)" = "Linux" ]; then
    CURRENT_MAP_COUNT=$(sysctl -n vm.max_map_count 2>/dev/null || echo "0")
    if [ "$CURRENT_MAP_COUNT" -lt 262144 ]; then
        echo "  ⚠ vm.max_map_count=$CURRENT_MAP_COUNT (建议 >= 262144)"
        echo "    FriendB 使用 mmap，建议执行:"
        echo "    sudo sysctl -w vm.max_map_count=262144"
        echo "    永久生效: echo 'vm.max_map_count=262144' | sudo tee -a /etc/sysctl.conf"
    else
        echo "  ✓ vm.max_map_count=$CURRENT_MAP_COUNT"
    fi
else
    echo "  - 非 Linux 系统，跳过内核参数检查"
fi

echo ""
echo "=========================================="
echo "  初始化完成！"
echo ""
echo "  本地运行:"
echo "    1. 编辑 .env 填入配置"
echo "    2. go mod download"
echo "    3. go build -o ymlink-server ./apps/server/"
echo "    4. source .env && ./ymlink-server"
echo ""
echo "  Docker 运行:"
echo "    docker-compose up -d"
echo "=========================================="
