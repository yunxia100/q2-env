# ============================================================
# YMLink-Q2 Docker 镜像 (含全部补丁)
# 使用 Go overlay 机制在编译时注入补丁
# ============================================================

FROM golang:1.22.4-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# 先拷贝 go.mod/go.sum 下载依赖 (利用 Docker 缓存层)
COPY ymlink-q2-new-master/go.mod ymlink-q2-new-master/go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 拷贝源码
COPY ymlink-q2-new-master/ ./

# 拷贝全部补丁文件 (保持目录结构)
COPY ymlink-q2-env/patches/ /patches/

# 生成 overlay.json 并编译
RUN set -e && \
    echo '{"Replace":{' > /tmp/overlay.json && \
    first=true && \
    for pf in $(find /patches -name "*.go" -type f | sort); do \
        rel="${pf#/patches/}"; \
        if [ "$first" = "true" ]; then first=false; else echo ',' >> /tmp/overlay.json; fi; \
        printf '"/build/%s":"%s"' "$rel" "$pf" >> /tmp/overlay.json; \
    done && \
    echo '}}' >> /tmp/overlay.json && \
    cat /tmp/overlay.json && \
    CGO_ENABLED=1 GOOS=linux go build -overlay=/tmp/overlay.json -o ymlink-server ./apps/server/

# ==================== 运行阶段 ====================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /build/ymlink-server .

# 前端静态文件
COPY ymlink-q2-new-master/apps/server/html ./html

# ip2region 数据库
COPY ymlink-q2-env/ip2region.xdb ./data/ip2region/ip2region.xdb

# 创建运行时目录
RUN mkdir -p data/friendb data/ip2region log \
    file/task file/material file/message \
    file/usedb file/qzonedb file/materialdb file/realinfodb \
    file/android_pack file/ios_pack file/ini_pack file/login && \
    touch data/friendb/friends.friendb

EXPOSE 8080

CMD ["./ymlink-server"]
