package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/hashicorp/yamux"
)

const (
	// 改成你的域名或IP，不要加 https://
	serverAddr = "host.com:7000"

	// 本地服务的地址 (例如公司内网的 API 服务器)
	localAddr = "127.0.0.1:8501"
)

func main() {
	fmt.Println("客户端启动...")
	fmt.Printf("目标服务器: %s\n本地转发服务: %s\n", serverAddr, localAddr)

	// 永久循环，确保断线后能自动重连
	for {
		connectServer()

		fmt.Println("!!! 连接断开或失败，5秒后尝试重连 !!!")
		time.Sleep(5 * time.Second)
	}
}

func connectServer() {
	// 配置 TLS
	// InsecureSkipVerify: true 是因为我们用的是自签证书，必须跳过验证
	tlsConf := &tls.Config{InsecureSkipVerify: true}

	fmt.Println("正在连接服务器 (TLS)...")
	conn, err := tls.Dial("tcp", serverAddr, tlsConf)
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}

	// 配置 Yamux 心跳 (关键：必须和服务端匹配)
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.KeepAliveInterval = 10 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 10 * time.Second

	// 建立客户端 Session
	session, err := yamux.Client(conn, yamuxConfig)
	if err != nil {
		fmt.Println("Session 创建失败:", err)
		conn.Close()
		return
	}
	fmt.Println("==> 成功连接到服务器！隧道已打通。")

	// 阻塞等待服务器发来的请求
	for {
		// Accept() 会阻塞，直到有新的流建立，或者连接断开
		stream, err := session.Accept()
		if err != nil {
			fmt.Println("隧道连接断开:", err)
			// 返回主函数，触发重连
			return
		}

		// 处理这个数据流
		go handleStream(stream)
	}
}

func handleStream(stream net.Conn) {
	defer stream.Close()

	// 连接本地服务 (普通 TCP 连接)
	localConn, err := net.Dial("tcp", localAddr)
	if err != nil {
		// 如果连不上本地服务，直接关闭这个流，不要让服务端傻等
		fmt.Println("无法连接本地服务:", err)
		return
	}
	defer localConn.Close()

	// 双向转发
	go io.Copy(localConn, stream)
	io.Copy(stream, localConn)
}
