#!/bin/bash
# ============================================================
#  将 q2-env 推送到 GitHub 仓库
#  在 Mac 本地终端运行: cd ~/Desktop/q2/ymlink-q2-env && bash git-push.sh
# ============================================================

set -e

REPO_URL="git@github.com:yunxia100/q2-env.git"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  推送 q2-env 到 GitHub"
echo "  仓库: $REPO_URL"
echo "=========================================="

# 检查是否已经是 git 仓库
if [ ! -d ".git" ]; then
    echo ""
    echo "[1/4] 初始化 Git 仓库"
    git init
    git branch -M main
    echo "  ✓ Git 仓库初始化完成"
else
    echo ""
    echo "[1/4] Git 仓库已存在"
    echo "  ✓ 跳过初始化"
fi

# 添加 remote
echo ""
echo "[2/4] 配置远程仓库"
if git remote get-url origin &>/dev/null 2>&1; then
    CURRENT_URL=$(git remote get-url origin)
    if [ "$CURRENT_URL" != "$REPO_URL" ]; then
        git remote set-url origin "$REPO_URL"
        echo "  ✓ 更新 remote: $REPO_URL"
    else
        echo "  ✓ remote 已配置: $REPO_URL"
    fi
else
    git remote add origin "$REPO_URL"
    echo "  ✓ 添加 remote: $REPO_URL"
fi

# 添加文件
echo ""
echo "[3/4] 添加文件到暂存区"

git add -A
echo "  文件状态:"
git status --short | head -30
CHANGE_COUNT=$(git status --short | wc -l | tr -d ' ')
echo "  ✓ 共 $CHANGE_COUNT 个变更"

# 提交并推送
echo ""
echo "[4/4] 提交并推送"

if [ "$CHANGE_COUNT" -gt "0" ]; then
    COMMIT_MSG="deploy: update patches and deploy scripts ($(date '+%Y-%m-%d %H:%M'))"
    git commit -m "$COMMIT_MSG"
    echo "  ✓ 提交完成: $COMMIT_MSG"

    echo ""
    echo "  推送到 GitHub ..."
    git push -u origin main --force
    echo "  ✓ 推送完成"
else
    echo "  没有变更需要提交"
fi

echo ""
echo "=========================================="
echo "  完成！仓库: https://github.com/yunxia100/q2-env"
echo ""
echo "  在新 Ubuntu 服务器上一键部署:"
echo "    1. 在服务器上创建目录: mkdir -p /root/q2"
echo "    2. 上传源码到服务器:   scp -r ymlink-q2-new-master root@IP:/root/q2/"
echo "    3. 在服务器上运行:"
echo "       cd /root/q2"
echo "       git clone https://github.com/yunxia100/q2-env.git"
echo "       bash q2-env/ubuntu-deploy.sh"
echo "=========================================="
