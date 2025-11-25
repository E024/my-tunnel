package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync" // 引入锁机制
	"time"

	"github.com/hashicorp/yamux"
)

const (
	tunnelPort = ":7000"
	httpPort   = ":8080"
	certFile   = "server.crt"
	keyFile    = "server.key"
)

var (
	// 使用读写锁保护 session，防止并发读写冲突
	sessionMutex sync.RWMutex
	session      *yamux.Session
)

func main() {
	go startTunnelServer()
	startHttpProxy()
}

func startTunnelServer() {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		panic("证书加载失败: " + err.Error())
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", tunnelPort, tlsConfig)
	if err != nil {
		panic("隧道启动失败: " + err.Error())
	}
	fmt.Println("==> [服务端] TLS 安全隧道启动成功！监听端口", tunnelPort)

	yamuxConfig := yamux.DefaultConfig()
	// 保持心跳配置，这很重要
	yamuxConfig.KeepAliveInterval = 10 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 10 * time.Second

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept Error:", err)
			continue
		}
		fmt.Println("==> [服务端] 检测到新的客户端连接:", conn.RemoteAddr())

		// 初始化新会话
		newSession, err := yamux.Server(conn, yamuxConfig)
		if err != nil {
			fmt.Println("Session 建立失败:", err)
			conn.Close()
			continue
		}

		// --- 【关键修改】加锁并清理旧会话 ---
		sessionMutex.Lock()
		if session != nil {
			// 如果之前有旧连接，强制关闭它，确保资源释放
			// 忽略关闭时的错误，因为旧连接可能已经断了
			session.Close()
			fmt.Println("--- [服务端] 已清理旧的僵尸会话 ---")
		}
		// 赋值为最新的会话
		session = newSession
		sessionMutex.Unlock()
		// ----------------------------------

		fmt.Println("==> [服务端] 隧道建立完毕，准备就绪！")
	}
}

func startHttpProxy() {
	listener, err := net.Listen("tcp", httpPort)
	if err != nil {
		panic("HTTP 代理启动失败: " + err.Error())
	}
	fmt.Println("==> [服务端] HTTP 代理启动成功！访问入口: http://你的IP" + httpPort)

	for {
		userConn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleUserRequest(userConn)
	}
}

func handleUserRequest(userConn net.Conn) {
	defer userConn.Close()

	// --- 【关键修改】加读锁获取 Session ---
	sessionMutex.RLock()
	currentSession := session
	sessionMutex.RUnlock()
	// ------------------------------------

	// 检查获取到的 session 是否有效
	if currentSession == nil || currentSession.IsClosed() {
		fmt.Fprintln(userConn, "Error: Tunnel Client is not connected.")
		return
	}

	// 打开流
	stream, err := currentSession.Open()
	if err != nil {
		// 这里虽然报错，但在重连瞬间是正常的
		fmt.Println("无法打开隧道流:", err)
		return
	}
	defer stream.Close()

	go io.Copy(userConn, stream)
	io.Copy(stream, userConn)
}
