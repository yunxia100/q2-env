#!/bin/bash
set -e
REPO_URL="git@github.com:yunxia100/q2-env.git"
cd "$(cd "$(dirname "$0")" && pwd)"

[ ! -d ".git" ] && { git init; git branch -M main; }
git remote get-url origin &>/dev/null 2>&1 && git remote set-url origin "$REPO_URL" || git remote add origin "$REPO_URL"
git add -A
COUNT=$(git status --short | wc -l | tr -d ' ')
[ "$COUNT" -gt "0" ] && { git commit -m "update $(date '+%Y-%m-%d %H:%M')"; git push -u origin main --force; } || echo "no changes"
