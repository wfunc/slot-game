# 多阶段构建：使用官方Go镜像作为构建环境
FROM golang:1.23-alpine AS builder

# 安装基本工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o slot-game ./cmd/server

# 运行阶段：使用精简镜像
FROM alpine:latest

# 安装基本运行时依赖
RUN apk --no-cache add ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非root用户
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S app -G app

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/slot-game .

# 复制配置文件（如果存在）
COPY --from=builder /app/config*.yaml ./

# 切换到非root用户
USER app

# 暴露端口
EXPOSE 8080
EXPOSE 9090

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./slot-game"]