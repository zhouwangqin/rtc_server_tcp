package src

import (
	"log"
	"sync"
)

// Room 房间对象
type Room struct {
	id         string
	peers      map[string]*Peer
	peersMutex sync.Mutex
}

// NewRoom 新建Room对象
func NewRoom(rid string) *Room {
	room := &Room{
		id:    rid,
		peers: make(map[string]*Peer),
	}
	return room
}

// ID 返回id
func (room *Room) ID() string {
	return room.id
}

// AddPeer 新增peer
func (room *Room) AddPeer(peer *Peer) {
	uid := peer.ID()
	// 删除老peer
	room.DelPeer(uid)
	// 添加新peer
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	room.peers[uid] = peer
}

// DelPeer 删除peer
func (room *Room) DelPeer(uid string) {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	if room.peers[uid] != nil {
		room.peers[uid].Close()
		delete(room.peers, uid)
	}
}

// GetPeer 获取peer
func (room *Room) GetPeer(uid string) *Peer {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	if peer, ok := room.peers[uid]; ok {
		return peer
	}
	return nil
}

// GetPeers 获取peers
func (room *Room) GetPeers() map[string]*Peer {
	return room.peers
}

// MapPeers 遍历所有的peer
func (room *Room) MapPeers(fn func(string, *Peer)) {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	for uid, peer := range room.peers {
		fn(uid, peer)
	}
}

// NotifyWithUid 通知房间指定人
func (room *Room) NotifyWithUid(uid, method string, data map[string]interface{}) {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	for id, peer := range room.peers {
		if id == uid {
			peer.Notify(method, data)
			break
		}
	}
}

// NotifyWithoutUid 通知房间里面其他人
func (room *Room) NotifyWithoutUid(fuid, method string, data map[string]interface{}) {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	for uid, peer := range room.peers {
		if uid != fuid {
			peer.Notify(method, data)
		}
	}
}

// NotifyAll 通知房间里面所有人
func (room *Room) NotifyAll(method string, data map[string]interface{}) {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	for _, peer := range room.peers {
		peer.Notify(method, data)
	}
}

// Close 关闭room
func (room *Room) Close() {
	room.peersMutex.Lock()
	defer room.peersMutex.Unlock()
	log.Printf("Close Room rid=%s", room.id)
	for _, peer := range room.peers {
		peer.Close()
	}
}
