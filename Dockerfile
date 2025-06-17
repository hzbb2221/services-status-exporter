# 构建阶段
FROM golang:1.23-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum文件
COPY go.mod ./
COPY go.sum ./

# 下载依赖
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o services-status-exporter .

# 运行阶段
FROM alpine:3.18

WORKDIR /app

# 安装ca-certificates，用于HTTPS请求
RUN apk --no-cache add ca-certificates

# 从构建阶段复制二进制文件
COPY --from=builder /app/services-status-exporter .

# 创建配置目录
RUN mkdir -p /app/web

# 暴露端口
EXPOSE 9112

# 设置健康检查
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --spider -q http://localhost:9112/metrics || exit 1

# 运行应用
ENTRYPOINT ["/app/services-status-exporter"]
CMD ["-web-dir", "/app/web", "-addr", ":9112"]