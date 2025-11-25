package main

import (
	"fmt"
	"net"
)

func main() {
	// 改成你的服务器 IP
	serverAddr, _ := net.ResolveUDPAddr("udp", "my2025.rqey.com:7000")
	conn, _ := net.DialUDP("udp", nil, serverAddr)
	defer conn.Close()

	conn.Write([]byte("hello udp"))
	fmt.Println("UDP 包已发送")
}
