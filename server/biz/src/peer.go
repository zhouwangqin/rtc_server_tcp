package src

import (
	"encoding/json"
	"fmt"
)

type Transcation struct {
	id     int
	accept AcceptFunc
	reject RejectFunc
}

type Peer struct {
	emit  *Emitter
	id    string
	tcp   *TcpSocket
	trans map[int]*Transcation
}

func NewPeer(id string, tcp *TcpSocket) *Peer {
	peer := new(Peer)
	peer.emit = NewEmitter()
	peer.id = id
	peer.tcp = tcp
	peer.trans = make(map[int]*Transcation)
	peer.tcp.emit.On("message", peer.handleMessage)
	peer.tcp.emit.On("error", func(code int, err string) {
		peer.emit.Emit("error", code, err)
	})
	return peer
}

func (peer *Peer) ID() string {
	return peer.id
}

func (peer *Peer) Work() {
	peer.tcp.Read()
}

func (peer *Peer) Close() {
	peer.tcp.Close()
}

func (peer *Peer) Request(method string, data map[string]interface{}, success AcceptFunc, reject RejectFunc) {
	id := GenerateRandomNumber()
	request := &Request{
		Request: true,
		Id:      id,
		Method:  method,
		Data:    data,
	}

	str, err := json.Marshal(request)
	if err != nil {
		return
	}

	transcation := &Transcation{
		id:     id,
		accept: success,
		reject: reject,
	}

	peer.trans[id] = transcation
	peer.tcp.Send(string(str))
}

func (peer *Peer) Notify(method string, data map[string]interface{}) {
	notification := &Notification{
		Notification: true,
		Method:       method,
		Data:         data,
	}

	str, err := json.Marshal(notification)
	if err != nil {
		return
	}

	fmt.Printf("Send notification [%s]\n", method)
	peer.tcp.Send(string(str))
}

func (peer *Peer) handleMessage(message []byte) {
	data := make(map[string]interface{})
	err := json.Unmarshal(message, &data)
	if err != nil {
		fmt.Printf("handleMessage Unmarshal err => %v", err)
		return
	}
	if data["request"] != nil {
		peer.handleRequest(data)
	} else if data["response"] != nil {
		peer.handleResponse(data)
	} else if data["notification"] != nil {
		peer.handleNotification(data)
	}
}

func (peer *Peer) handleRequest(request map[string]interface{}) {
	fmt.Printf("Handle request [%s]", request["method"])
	accept := func(data map[string]interface{}) {
		response := &Response{
			Response: true,
			Ok:       true,
			Id:       int(request["id"].(float64)),
			Data:     data,
		}
		str, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("accept Marshal %v", err)
			return
		}

		peer.tcp.Send(string(str))
	}

	reject := func(errorCode int, errorReason string) {
		response := &ResponseError{
			Response:    true,
			Ok:          false,
			Id:          int(request["id"].(float64)),
			ErrorCode:   errorCode,
			ErrorReason: errorReason,
		}
		str, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("reject Marshal %v", err)
			return
		}

		peer.tcp.Send(string(str))
	}

	peer.emit.Emit("request", request, accept, reject)
}

func (peer *Peer) handleResponse(response map[string]interface{}) {
	id := int(response["id"].(float64))
	transcation := peer.trans[id]
	if transcation == nil {
		fmt.Printf("received response does not match any sent request [id:%d]", id)
		return
	}

	if response["ok"] != nil && response["ok"] == true {
		transcation.accept(response["data"].(map[string]interface{}))
	} else {
		transcation.reject(int(response["errorCode"].(float64)), response["errorReason"].(string))
	}

	delete(peer.trans, id)
}

func (peer *Peer) handleNotification(notification map[string]interface{}) {
	peer.emit.Emit("notification", notification)
}
