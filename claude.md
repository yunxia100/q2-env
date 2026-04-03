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

## 前端访问

编译好的前端 `html/` 内 API 地址指向远程服务器，本地开发需加参数：

```
http://localhost:8080/?url=localhost:8080
```

或使用 `one-click-start.sh` 启动前端开发服务器（端口 3000，已自动补丁 API 地址）。
