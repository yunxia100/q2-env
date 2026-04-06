#!/bin/bash
# ============================================================
# YMLink-Q2 Mac 本地启动脚本
# 用法: chmod +x start-mac.sh && ./start-mac.sh
# ============================================================

cd "$(dirname "$0")/../ymlink-q2-new-master" || exit 1

# -------- 环境变量 --------
export SYSTEM_MODE=debug
export SYSTEM_LOG_LEVEL=debug
export SYSTEM_MEM_LIMIT=0
export SYSTEM_CPU_LIMIT=4
export SYSTEM_ROBOT_THREAD_LIMIT=10
export SYSTEM_ROBOT_MESSAGE_LIMIT=100
export SYSTEM_DBLOAD=true

export HTTP_SERVER1_URL=:8080
export HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=ymlink-jwt-secret-key-2024
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

export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb
export FRIENDB1_TOTAL=100000

export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb

export DRIVE_MAPPING=server1#androidQQ#8.9.80#http://localhost:8098
export DRIVE_TIMEOUT=30
export DRIVE_MAX_CONN=50

export MOBILE_PORT=16100

# -------- 日志文件 --------
# 日志同时输出到控制台和 ./log/ymlink-q2.log
echo "=========================================="
echo "  YMLink-Q2 Starting..."
echo "  HTTP: http://localhost:8080"
echo "  Log:  ./log/ymlink-q2.log"
echo "  Ctrl+C to stop"
echo "=========================================="

./ymlink-server
