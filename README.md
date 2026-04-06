# YMLink-Q2 环境配置文件夹

## 文件说明

```
ymlink-q2-env/
├── .env.example              # 环境变量模板（本地运行用）
├── .env.docker               # Docker Compose 专用环境变量
├── docker-compose.yml        # Docker 编排（MongoDB + InfluxDB + 驱动 + Server）
├── Dockerfile                # 服务端构建镜像
├── setup.sh                  # 一键初始化脚本
├── influx-init.sh            # InfluxDB Bucket 初始化
├── mongo-init/
│   └── init-collections.js   # MongoDB 集合和索引自动创建
├── drive-server/             # ★ 底层驱动目录（将驱动文件放这里）
│   └── README.md             # 驱动放置说明
└── README.md
```

## 快速开始

### 方式一：Docker（推荐）

```bash
cd ymlink-q2-env

# 1. 初始化环境（创建目录 + 下载 ip2region.xdb）
chmod +x setup.sh && ./setup.sh

# 2. 将底层驱动程序放入 drive-server/ 目录

# 3. 编辑 docker-compose.yml，取消 drive-server 服务的注释，
#    修改 command 为你的驱动可执行文件名

# 4. 启动
docker-compose up -d
```

### 方式二：本地运行

```bash
cd ymlink-q2-env
chmod +x setup.sh && ./setup.sh
vim .env                            # 编辑配置

# 先启动底层驱动
cd drive-server && ./你的驱动程序 &

# 再启动主服务
cd ../ymlink-q2-new-master
go mod download
go build -o ymlink-server ./apps/server/
export $(grep -v '^#' ../ymlink-q2-env/.env | xargs)
./ymlink-server
```

## 依赖服务

| 服务 | 默认端口 | 说明 |
|------|---------|------|
| **底层驱动 (Drive)** | **8098** | **QQ协议通信，WebSocket连接，必须先启动** |
| MongoDB 6.0 | 27017 | 主数据库 |
| InfluxDB 2.7 | 8086 | 时序数据库（realtime + history） |

## 底层驱动配置

驱动通过 `DRIVE_MAPPING` 环境变量配置：

```
格式: 硬件类型#软件类型#软件版本#驱动URL
示例: server1#androidQQ#8.9.80#http://localhost:8098
多个: server1#androidQQ#8.9.80#http://10.0.0.1:8098,server2#iOSQQ#9.0.0#http://10.0.0.2:8098
```

主服务自动将 `http://` 转为 `ws://` 连接驱动的 `/websocket` 端点。

## MongoDB 集合（21个）

自动由 `mongo-init/init-collections.js` 创建，含建议索引。

## InfluxDB Bucket

| Bucket | 保留策略 | 说明 |
|--------|---------|------|
| realtime | 7天 | 实时监控数据 |
| history | 365天 | 历史统计数据 |
