#!/bin/bash
cd /Users/df/Desktop/q2/ymlink-q2-new-master

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

exec ./ymlink-server >> ./log/ymlink-q2.log 2>&1
