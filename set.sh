#!/bin/bash
# ============================================================
# 设置机器人"加我为好友的方式"
#
# 用法:
#   chmod +x set.sh
#   ./set.sh                          # 使用默认参数
#   ./set.sh -q 2137274756 -d XCNFu   # 指定QQ号和设备ID
#   ./set.sh -q 2137274756 -d XCNFu -t 1  # 指定验证方式
#   ./set.sh -q 2137274756 -d XCNFu -s http://8.130.31.166:8098
#
# 验证方式 (-t):
#   0 = 允许任何人
#   1 = 需要验证信息 (默认)
#   2 = 禁止加好友
#   3 = 需要回答问题
# ============================================================

set -e

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

# ============ 默认参数 ============
QQ_UIN="2137274756"
DEVICE_ID="XCNFu"
DRIVE_URL="http://8.130.31.166:8098"
AUTH_TYPE=1   # 需要验证信息

# ============ 解析命令行参数 ============
while getopts "q:d:s:t:h" opt; do
    case $opt in
        q) QQ_UIN="$OPTARG" ;;
        d) DEVICE_ID="$OPTARG" ;;
        s) DRIVE_URL="$OPTARG" ;;
        t) AUTH_TYPE="$OPTARG" ;;
        h)
            echo "用法: $0 [-q QQ号] [-d 设备ID] [-s 驱动地址] [-t 验证方式]"
            echo ""
            echo "参数:"
            echo "  -q  QQ号        (默认: $QQ_UIN)"
            echo "  -d  设备ID      (默认: $DEVICE_ID)"
            echo "  -s  驱动地址    (默认: $DRIVE_URL)"
            echo "  -t  验证方式    (默认: 1)"
            echo ""
            echo "验证方式:"
            echo "  0 = 允许任何人"
            echo "  1 = 需要验证信息"
            echo "  2 = 禁止加好友"
            echo "  3 = 需要回答问题"
            exit 0
            ;;
        *) echo "无效参数, 使用 -h 查看帮助"; exit 1 ;;
    esac
done

# ============ 验证方式名称 ============
case $AUTH_TYPE in
    0) TYPE_NAME="允许任何人" ;;
    1) TYPE_NAME="需要验证信息" ;;
    2) TYPE_NAME="禁止加好友" ;;
    3) TYPE_NAME="需要回答问题" ;;
    *) TYPE_NAME="未知($AUTH_TYPE)" ;;
esac

echo ""
echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo -e "${CYAN}  设置加我为好友的方式${NC}"
echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo -e "  QQ号:     ${GREEN}${QQ_UIN}${NC}"
echo -e "  设备ID:   ${GREEN}${DEVICE_ID}${NC}"
echo -e "  驱动地址: ${GREEN}${DRIVE_URL}${NC}"
echo -e "  验证方式: ${GREEN}${AUTH_TYPE} (${TYPE_NAME})${NC}"
echo ""

# ============ 设置好友验证方式 (通过OIDB底层协议) ============
echo -e "${YELLOW}[1/1] 通过OIDB协议设置好友验证方式为: ${TYPE_NAME}...${NC}"

SET_URL="${DRIVE_URL}/device/oidb587x75?objid=${DEVICE_ID}"
SET_BODY="{\"uint32_allow\":${AUTH_TYPE}}"

echo -e "  请求: POST ${SET_URL}"
echo -e "  Body: ${SET_BODY}"

# 驱动代理不设Content-Encoding头, 需手动gunzip
TMPFILE=$(mktemp)
SET_HTTP_CODE=$(curl -s -o "$TMPFILE" -w "%{http_code}" \
    -X POST "$SET_URL" \
    -H "Content-Type: application/json" \
    -d "$SET_BODY" 2>/dev/null) || true

# 尝试gunzip, 如果不是gzip就直接读取
SET_BODY_RESP=$(gunzip < "$TMPFILE" 2>/dev/null || cat "$TMPFILE")
rm -f "$TMPFILE"

echo -e "  响应: ${SET_BODY_RESP}"
echo ""

if [ "$SET_HTTP_CODE" = "200" ]; then
    # 检查OIDB响应
    if echo "$SET_BODY_RESP" | grep -q '"ErrorCode":0'; then
        echo -e "  ${GREEN}✓ 设置成功! 好友验证方式已设为: ${TYPE_NAME}${NC}"
    elif echo "$SET_BODY_RESP" | grep -q '"ActionStatus":"OK"'; then
        echo -e "  ${GREEN}✓ 设置成功! 好友验证方式已设为: ${TYPE_NAME}${NC}"
    else
        echo -e "  ${RED}✗ OIDB返回错误, 请检查响应内容${NC}"
        echo -e "  ${YELLOW}提示: 如果sKey过期, 请先让机器人重新上线${NC}"
    fi
else
    echo -e "  ${RED}✗ 请求失败 (HTTP ${SET_HTTP_CODE})${NC}"
fi

echo ""
echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo -e "${CYAN}  完成${NC}"
echo -e "${CYAN}══════════════════════════════════════════${NC}"
echo ""
