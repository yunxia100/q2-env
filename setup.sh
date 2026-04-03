#!/bin/bash
set -e
for d in data/friendb data/ip2region file/task file/material file/message file/usedb file/qzonedb file/materialdb file/realinfodb file/android_pack file/ios_pack file/ini_pack file/login; do
    mkdir -p "$d"
done
[ ! -f "data/ip2region/ip2region.xdb" ] && curl -fSL -o "data/ip2region/ip2region.xdb" "https://raw.githubusercontent.com/lionsoul2014/ip2region/master/data/ip2region.xdb" 2>/dev/null || true
[ ! -f "data/friendb/friends.friendb" ] && touch data/friendb/friends.friendb
[ -f ".env.example" ] && [ ! -f ".env" ] && cp .env.example .env
echo "setup done"
