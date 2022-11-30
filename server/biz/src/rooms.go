package src

import "sync"

// Rooms 房间管理对象
type Rooms struct {
	roomMap    map[string]*Room
	roomsMutex sync.Mutex
}

// NewRooms 新建Rooms对象
func NewRooms() *Rooms {
	rooms := &Rooms{
		roomMap: make(map[string]*Room),
	}
	return rooms
}

// AddRoom 增加room
func (rooms *Rooms) AddRoom(rid string) *Room {
	room := rooms.GetRoom(rid)
	if room == nil {
		// 创建新room
		room = NewRoom(rid)
		// 增加新room
		rooms.roomsMutex.Lock()
		defer rooms.roomsMutex.Unlock()
		rooms.roomMap[rid] = room
	}
	return room
}

// DelRoom 删除room
func (rooms *Rooms) DelRoom(rid string) {
	rooms.roomsMutex.Lock()
	defer rooms.roomsMutex.Unlock()
	if rooms.roomMap[rid] != nil {
		rooms.roomMap[rid].Close()
		delete(rooms.roomMap, rid)
	}
}

// GetRoom 获取room
func (rooms *Rooms) GetRoom(rid string) *Room {
	rooms.roomsMutex.Lock()
	defer rooms.roomsMutex.Unlock()
	if room, ok := rooms.roomMap[rid]; ok {
		return room
	}
	return nil
}

// GetRooms 获取rooms
func (rooms *Rooms) GetRooms() map[string]*Room {
	return rooms.roomMap
}

// NotifyWithUid 通知房间指定人
func (rooms *Rooms) NotifyWithUid(rid, uid, method string, data map[string]interface{}) {
	room := rooms.GetRoom(rid)
	if room != nil {
		room.NotifyWithUid(uid, method, data)
	}
}

// NotifyWithoutUid 通知房间里面其他人
func (rooms *Rooms) NotifyWithoutUid(rid, fuid, method string, data map[string]interface{}) {
	room := rooms.GetRoom(rid)
	if room != nil {
		room.NotifyWithoutUid(fuid, method, data)
	}
}

// NotifyAll 通知房间里面所有人
func (rooms *Rooms) NotifyAll(rid, method string, data map[string]interface{}) {
	room := rooms.GetRoom(rid)
	if room != nil {
		room.NotifyAll(method, data)
	}
}
