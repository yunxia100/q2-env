# YMLink-Q2 项目规则

## 核心原则：源码目录只读

`ymlink-q2-new-master/` 目录下的所有文件和代码**严禁修改**，包括但不限于：

- 不能修改任何 `.go` 源码文件
- 不能修改文件时间戳
- 不能添加、删除或重命名文件
- 不能修改 `go.mod`、`go.sum`
- 不能修改 `.web/` 前端源码
- 不能修改 `apps/server/html/` 编译产物

所有修改必须通过**补丁机制**在 `ymlink-q2-env/` 目录内完成。

## 补丁机制

### Go 后端补丁

使用 Go 的 `-overlay` 编译参数，在编译/运行时替换文件，不动源码：

- 补丁文件位置：`ymlink-q2-env/patches/`
- 映射配置：`ymlink-q2-env/patches/overlay.json`（脚本自动生成）
- 现有补丁：
  - `patches/apps/server/main.go` — 业务日志写入文件
  - `patches/plugin/plugin.http.server.go` — Gin 请求日志 + panic 恢复

### Vue 前端补丁

启动前自动覆盖，停止时自动还原：

- 补丁文件位置：`ymlink-q2-env/patches/.web/`
- 现有补丁：
  - `patches/.web/setting.ts` — API 地址改为 localhost:8080

## 可修改的配置

### 底层驱动配置

底层驱动（Drive Server）的配置在一键脚本中的环境变量 `DRIVE_MAPPING`：

```bash
export DRIVE_MAPPING='server1#iOSQQ#8.9.80#http://8.130.31.166:8098'
```

格式：`硬件名#软件名#版本号#WebSocket地址`，多个驱动用逗号分隔。

当前底层驱动运行在测试服务器：
- 服务器：`8.130.31.166`
- 进程：`qq_mini`（端口 8098 / 8097）
- 类型：iOS QQ 驱动

驱动程序如需本地运行，放在：`ymlink-q2-env/drive-server/`

### 数据库配置

MongoDB 和 InfluxDB 的连接信息在一键脚本的环境变量中，可按需修改：

- `MONGO1_URL` — MongoDB 地址（不含 `mongodb://` 前缀，代码会自动拼接）
- `MONGO1_USERNAME` / `MONGO1_PASSWORD` — MongoDB 认证
- `INFLUX1_URL` — InfluxDB 地址
- `INFLUX1_TOKEN` — InfluxDB 访问令牌

### 其他可修改项

`ymlink-q2-env/` 目录下的所有文件均可修改：

- `one-click-start.sh` — 开发模式启动脚本
- `one-click-deploy.sh` — 编译部署脚本
- `one-click-docker.sh` — Docker 全容器部署脚本
- `docker-compose.yml` — Docker Compose 配置
- `Dockerfile` — Docker 镜像构建
- `.env.example` / `.env.docker` — 环境变量模板
- `ip2region.xdb` — IP 地理定位数据库

## 目录结构

```
├── ymlink-q2-new-master/    ← 只读！从 gitee 拉取的源码
└── ymlink-q2-env/           ← 可修改，所有配置和脚本在这里
    ├── one-click-start.sh        开发模式 (go run)
    ├── one-click-deploy.sh       编译部署 (go build)
    ├── one-click-docker.sh       Docker 全容器
    ├── patches/                  补丁目录
    │   ├── apps/server/main.go
    │   ├── plugin/plugin.http.server.go
    │   └── .web/setting.ts
    ├── drive-server/             底层驱动程序放这里
    ├── ip2region.xdb             IP 库
    ├── docker-compose.yml
    ├── Dockerfile
    └── mongo-init/               MongoDB 初始化脚本
```

## 编译部署工作流（必须严格遵守）

仓库 `ymlink-q2-env` (git@github.com:yunxia100/q2-env.git) 只提交编译产物，不提交源码。

### 仓库中 git 跟踪的文件

- `q2-env-patch` — 编译好的 server 二进制（linux/amd64, 静态链接）
- `web-env/` — 构建好的前端产物（index.html + assets/ + images/）
- 配置文件：`.env.docker`, `docker-compose.yml`, `ip2region.xdb`, `mongo-init/`

### 仓库中 .gitignore 忽略的文件（不提交）

- `patches/` — Go/Vue/TS 源码补丁（本地开发用）
- `*.go`, `*.ts`, `*.vue`, `*.sh`, `*.md`, `Dockerfile`

### 编译步骤

每次修改补丁代码后，必须按以下步骤编译并提交：

#### 1. 合并源码 + 补丁到临时构建目录

```bash
cd ~/Desktop/q2
rm -rf _build
rsync -a --ignore-errors ymlink-q2-master/ _build/ 2>/dev/null

# 覆盖补丁文件（Go后端）
rsync -a ymlink-q2-env/patches/model/ _build/model/ 2>/dev/null
rsync -a ymlink-q2-env/patches/apps/ _build/apps/ 2>/dev/null
rsync -a ymlink-q2-env/patches/plugin/ _build/plugin/ 2>/dev/null
rsync -a ymlink-q2-env/patches/define/ _build/define/ 2>/dev/null
```

#### 2. 交叉编译 server 二进制（Mac → Linux）

```bash
cd ~/Desktop/q2/_build
GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -o ../ymlink-q2-env/q2-env-patch apps/server/*.go
```

#### 3. 构建前端（如有前端改动）

```bash
cd ~/Desktop/q2
rm -rf _build_web
rsync -a --ignore-errors ymlink-q2-master/.web/ _build_web/ 2>/dev/null
rsync -a ymlink-q2-env/patches/.web/ _build_web/

cd _build_web
npm install
npm run build

# 复制构建产物
cd ~/Desktop/q2/ymlink-q2-env
rm -rf web-env/assets web-env/images web-env/index.html
cp -r ~/Desktop/q2/_build_web/dist/assets web-env/assets
cp -r ~/Desktop/q2/_build_web/dist/images web-env/images
cp ~/Desktop/q2/_build_web/dist/index.html web-env/index.html
```

#### 4. 提交并推送

```bash
cd ~/Desktop/q2/ymlink-q2-env
git add q2-env-patch web-env/
git commit -m "feat/fix: 描述改动"
git push origin main
```

#### 5. 服务器部署

```bash
# SSH 到服务器
ssh root@8.130.31.166  # 密码 Yunmi200@2025

# 服务器上拉取并重启
# TODO: 部署脚本待完善（当前服务器用的是 /opt/ymlink-q2/ Docker 方式）
```

### 重要注意事项

- `ymlink-q2-master/` 是只读源码基线，绝对不能修改
- `ymlink-q2-new-master/` 是新版源码参考，也不能修改
- 所有改动只在 `ymlink-q2-env/patches/` 中进行
- 补丁文件命名：新增功能用 `patch_xxx.go` 前缀，修改已有文件直接同名覆盖
- 编译前必须合并到 `_build/` 临时目录，不能直接在源码目录编译
- server 二进制必须用 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 交叉编译

### 现有补丁清单

Go 后端补丁（`patches/`）：
- `patches/apps/server/main.go.patched` — 添加好友请求路由 + 禁用3小时自动退出
- `patches/apps/server/ctrler/patch_friend_notice_ctrler.go` — 好友请求控制器
- `patches/model/patch_friend_notice_model.go` — 好友请求模型方法
- `patches/apps/server/ctrler/ctrler.custservice.api.go` — 客服API扩展
- `patches/apps/server/ctrler/ctrler.robot.deal.message.go` — 机器人消息处理
- `patches/apps/server/ctrler/ctrler.task.deal.materialgreet.greet.go` — 素材打招呼
- `patches/apps/server/ctrler/ctrler.task.deal.qzoneremark.go` — 空间留痕
- `patches/model/model.robot.kernel.go` — 机器人内核模型
- `patches/model/model.robot.message.go` — 机器人消息模型
- `patches/plugin/plugin.http.server.go` — Gin请求日志+panic恢复
- `patches/define/` — 常量定义

Vue 前端补丁（`patches/.web/`）：
- `patches/.web/setting.ts` — API地址配置
- `patches/.web/src/page/platform/studio/account/robot.friend/index.func.vue` — 好友功能工具栏
- `patches/.web/src/page/platform/studio/account/robot.friend/index.func.friend.notice.vue` — 好友请求弹窗
- `patches/.web/src/page/platform/studio/account/robot.friend/index.func.friend.notice.ts` — 好友请求逻辑
- `patches/.web/src/page/robot.batch/account/index.table.vue` — 批量账号表格
- `patches/.web/src/page/robot.batch/account/index.table.slider.login.vue` — 滑块登录弹窗
- `patches/.web/src/page/robot.batch/account/index.ts` — 批量账号逻辑

### 测试服务器信息

- IP: `8.130.31.166`
- SSH: `root / Yunmi200@2025`
- 前端: `http://8.130.31.166:8088`
- 后端API: `http://8.130.31.166:16000`
- Drive API: `http://8.130.31.166:8098`
- 部署目录: `/opt/ymlink-q2/`（Docker方式）

## 前端访问

编译好的前端 `html/` 内 API 地址指向远程服务器，本地开发需加参数：

```
http://localhost:8080/?url=localhost:8080
```

或使用 `one-click-start.sh` 启动前端开发服务器（端口 3000，已自动补丁 API 地址）。
