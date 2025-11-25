go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

# 初始化模块
go mod init my-tunnel

# 下载依赖库 (Yamux)
go get github.com/hashicorp/yamux

# 设置目标系统为 Linux，架构为 64位
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -o server_linux ./server/main.go

# 编译 server 文件夹
go build -o server_linux ./server/main.go

# 设置回 Windows 环境
set GOOS=windows
set GOARCH=amd64
go build -o client.exe ./client/main.go

# 1. 生成私钥 (server.key)
openssl genrsa -out server.key 2048

# 2. 生成证书 (server.crt)
# 提示输入信息时，Common Name 填你的域名或随便填，其他可以直接回车
openssl req -new -x509 -key server.key -out server.crt -days 3650