package src

import (
	"fmt"
	"log"
	"net"
	"server/pkg/util"
	"strconv"
)

// err code
const (
	// ErrInvalidMethod ...
	ErrInvalidMethod = "method not found"
	// ErrInvalidData ...
	ErrInvalidData = "data not found"
)

// DefaultAccept 默认接受处理
func DefaultAccept(data map[string]interface{}) {
	log.Printf("tcp accept data => %v", data)
}

// DefaultReject 默认拒绝处理
func DefaultReject(errorCode int, errorReason string) {
	log.Printf("tcp reject errorCode => %v errorReason => %v", errorCode, errorReason)
}

// Start tcp server
func StartTcp(ip string, port uint16) {
	fmt.Println("tcp server start")
	url := ip + ":" + strconv.Itoa(int(port))
	lner, err := net.Listen("tcp", url)
	if err != nil {
		fmt.Printf("tcp server listen error = %v", err)
		return
	}

	for {
		conn, err := lner.Accept()
		if err != nil {
			fmt.Printf("server accept error = %v", err)
			break
		}
		go handleConnection(conn)
	}

	if lner != nil {
		lner.Close()
	}
	fmt.Println("tcp server stop")
}

func handleConnection(conn net.Conn) {
	tcp := NewTcpSocket(conn)
	peer := NewPeer("", tcp)

	handleRequest := func(request map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
		method := util.Val(request, "method")
		if method == "" {
			reject(-1, ErrInvalidMethod)
			return
		}

		data := request["data"]
		if data == nil {
			reject(-1, ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		handlerWebsocket(method, peer, msg, accept, reject)
	}

	handleNotification := func(notification map[string]interface{}) {
		method := util.Val(notification, "method")
		if method == "" {
			DefaultReject(-1, ErrInvalidMethod)
			return
		}

		data := notification["data"]
		if data == nil {
			DefaultReject(-1, ErrInvalidData)
			return
		}

		msg := data.(map[string]interface{})
		handlerWebsocket(method, peer, msg, DefaultAccept, DefaultReject)
	}

	handleClose := func(code int, err string) {
		peer.Close()
	}

	peer.emit.On("request", handleRequest)
	peer.emit.On("notification", handleNotification)
	peer.emit.On("error", handleClose)
	peer.emit.On("close", handleClose)
	peer.Work()
}
