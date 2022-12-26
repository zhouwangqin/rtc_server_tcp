package src

import (
	"errors"
	"fmt"
	"net"
	"sync"
)

type TcpSocket struct {
	emit  *Emitter
	conn  net.Conn
	mutex sync.Mutex
	close bool
}

func NewTcpSocket(conn net.Conn) *TcpSocket {
	tcp := new(TcpSocket)
	tcp.emit = NewEmitter()
	tcp.conn = conn
	tcp.close = false
	return tcp
}

func (tcp *TcpSocket) Read() {
	msg := make(chan []byte)
	stop := make(chan int)
	go tcp.DoRead(msg, stop)

	for {
		select {
		case message := <-msg:
			{
				tcp.emit.Emit("message", []byte(message))
			}
		case <-stop:
			return
		}
	}
}

// read thread
func (tcp *TcpSocket) DoRead(msg chan []byte, stop chan int) {
	for {
		if tcp.close {
			close(stop)
			return
		}

		// read json data len
		byLen := make([]byte, 2)
		recvLen, err := tcp.conn.Read(byLen)
		if (err != nil) || (recvLen != 2) {
			fmt.Printf("tcp recv len error = %v\n", err)
			close(stop)
			// exit
			if !tcp.close {
				tcp.emit.Emit("error", 101, "tcp recv len error")
			}
			return
		}

		// calc json data len
		nLen := uint32(byLen[0])
		nLen += (uint32(byLen[1]) << 8)
		fmt.Printf("tcp recv len = %d\n", nLen)

		if nLen > 4096 {
			close(stop)
			// exit
			if !tcp.close {
				tcp.emit.Emit("error", 102, "tcp recv len error")
			}
			return
		}

		// read json data
		nIndex := 0
		byData := make([]byte, nLen)
		for nIndex < int(nLen) {
			recvLen, err = tcp.conn.Read(byData[nIndex:])
			if err != nil {
				fmt.Printf("tcp recv data error = %v\n", err)
				close(stop)
				// exit
				if !tcp.close {
					tcp.emit.Emit("error", 103, "tcp recv data error")
				}
				return
			}
			nIndex += recvLen
		}

		fmt.Printf("tcp recv data = %s\n", string(byData))
		msg <- byData
	}
}

func (tcp *TcpSocket) Send(msg string) error {
	tcp.mutex.Lock()
	defer tcp.mutex.Unlock()
	if tcp.close {
		return errors.New("tcp write closed")
	}

	// add len 2 byte
	nLen := len(msg)
	byData := make([]byte, nLen+2)
	byData[0] = byte(nLen)
	byData[1] = byte(nLen >> 8)
	copy(byData[2:], []byte(msg))
	fmt.Printf("tcp send data = %s\n", msg)

	_, err := tcp.conn.Write(byData)
	if err != nil {
		return errors.New("tcp write fail")
	}
	return nil
}

func (tcp *TcpSocket) Close() {
	tcp.mutex.Lock()
	defer tcp.mutex.Unlock()
	if !tcp.close {
		tcp.close = true
		tcp.conn.Close()
	}
}
