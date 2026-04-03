#!/bin/bash
# ============================================================
# InfluxDB Bucket 初始化脚本
# 在 InfluxDB 首次启动后运行，创建所需的 Bucket
# ============================================================

set -e

INFLUX_URL="${INFLUX1_URL:-http://localhost:8086}"
INFLUX_TOKEN="${INFLUX1_TOKEN:-your-influx-token-change-this}"
INFLUX_ORG="${INFLUX1_ORG:-ymlink}"

echo "=========================================="
echo "  InfluxDB Bucket 初始化"
echo "  URL: $INFLUX_URL"
echo "  ORG: $INFLUX_ORG"
echo "=========================================="

# 创建 realtime bucket (实时数据)
echo "创建 bucket: realtime ..."
influx bucket create \
    --host "$INFLUX_URL" \
    --token "$INFLUX_TOKEN" \
    --org "$INFLUX_ORG" \
    --name "realtime" \
    --retention 7d \
    2>/dev/null && echo "  ✓ realtime 创建成功" || echo "  - realtime 已存在"

# 创建 history bucket (历史数据)
echo "创建 bucket: history ..."
influx bucket create \
    --host "$INFLUX_URL" \
    --token "$INFLUX_TOKEN" \
    --org "$INFLUX_ORG" \
    --name "history" \
    --retention 365d \
    2>/dev/null && echo "  ✓ history 创建成功" || echo "  - history 已存在"

echo ""
echo "✓ InfluxDB 初始化完成"
