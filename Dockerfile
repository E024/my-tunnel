# --- 第一阶段：编译环境 ---
FROM golang:alpine AS builder

WORKDIR /app

# 设置国内代理，加速依赖下载
#ENV GOPROXY=https://goproxy.cn,direct

# 复制所有文件 (包括 server 目录和证书)
COPY . .

# 初始化并下载依赖
# 如果没有 go.mod，这里会自动处理
RUN if [ ! -f go.mod ]; then go mod init my-tunnel; fi
RUN go mod tidy

# 编译服务端 (指定编译 server/main.go)
RUN go build -o server_bin ./server/main.go

# --- 第二阶段：运行环境 ---
FROM alpine:latest

WORKDIR /app

# 从 builder 阶段复制编译好的二进制文件
COPY --from=builder /app/server_bin .

# 复制证书文件 (必须有，否则 TLS 启动报错)
COPY server.crt server.key ./

# 赋予执行权限
RUN chmod +x server_bin

# 暴露端口
EXPOSE 7000/udp 8080/tcp

# 启动
CMD ["./server_bin"]