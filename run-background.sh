#!/bin/bash
BASE="$(cd "$(dirname "$0")" && pwd)"
WORK="$BASE/server-data"
BINARY="$BASE/q2-server"
cd "$WORK"

export SYSTEM_MODE=debug SYSTEM_LOG_LEVEL=debug SYSTEM_MEM_LIMIT=0 SYSTEM_CPU_LIMIT=4
export SYSTEM_THREAD_LIMIT=1 SYSTEM_MSG_LIMIT=100 SYSTEM_DBLOAD=true
export HTTP_SERVER1_URL=:8080 HTTP_SERVER1_MODE=debug
export HTTP_SERVER1_AUTH_PASSWORD=changeme HTTP_SERVER1_AUTH_TIMEOUT=604800
export HTTP_SERVER1_HTML_PATH=./html
export MONGO1_URL='localhost:27017/q2_db?authSource=admin' MONGO1_DATABASE=q2_db
export MONGO1_USERNAME=admin MONGO1_PASSWORD=admin
export INFLUX1_URL=http://localhost:8086 INFLUX1_ORG=q2org INFLUX1_TOKEN=q2-influx-token
export FRIENDB1_FILE_PATH=./data/friendb/friends.friendb FRIENDB1_TOTAL=100000
export IP2REGION1_FILE_PATH=./data/ip2region/ip2region.xdb
export DRIVE_MAPPING='server1#client#8.9.80#http://localhost:8098'
export DRIVE_TIMEOUT=30 DRIVE_MAX_CONN=50 MOBILE_PORT=16100

exec "$BINARY" >> "$BASE/server.log" 2>&1
