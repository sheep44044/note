# -----------------------------------------------------------------------------
# 1. Build Stage (构建层)
# 使用官方 Go 镜像编译代码，为了速度和安全
# -----------------------------------------------------------------------------
FROM golang:1.23-alpine AS builder

# 设置环境变量：启用模块化，设置代理以加快下载速度
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 设置工作目录
WORKDIR /app

# 先拷贝依赖文件，利用 Docker 缓存机制加速构建
COPY go.mod go.sum ./
RUN go mod download

# 拷贝源代码
COPY . .

# 编译 Go 程序
# -o main: 输出文件名为 main
# cmd/main.go: 你的入口文件路径
RUN go build -ldflags="-s -w" -o main cmd/main.go

# -----------------------------------------------------------------------------
# 2. Run Stage (运行层)
# 使用极小的 Alpine 镜像作为运行环境
# -----------------------------------------------------------------------------
FROM alpine:latest

# 安装基础证书（调用 HTTPS 接口如 OpenAI/豆包 必需）和时区数据
RUN apk --no-cache add ca-certificates tzdata

# 设置时区为上海
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从构建层只拷贝编译好的二进制文件和配置文件
COPY --from=builder /app/main .
# 如果你有静态资源（如 manifest, templates），也需要 COPY 进去
# COPY --from=builder /app/manifest ./manifest

# 暴露端口 (和你配置文件里的端口一致)
EXPOSE 8888

# 启动命令
CMD ["./main"]