package rtc

import (
	"server/server/sfu/conf"
	"sync"
	"time"

	"github.com/pion/webrtc/v2"
	"github.com/zhuanxin-sz/go-protoo/logger"
)

const (
	statCycle    = 5 * time.Second
	maxCleanSize = 100
)

var (
	stop         bool
	icePortStart uint16
	icePortEnd   uint16
	iceServers   []webrtc.ICEServer
	routers      map[string]*Router
	routersLock  sync.Mutex
	CleanRouter  chan string
)

// 初始化RTC
func InitRTC() {
	stop = false
	if len(conf.WebRTC.ICEPortRange) == 2 {
		icePortStart = conf.WebRTC.ICEPortRange[0]
		icePortEnd = conf.WebRTC.ICEPortRange[1]
	}

	iceServers = make([]webrtc.ICEServer, 0)
	for _, iceServer := range conf.WebRTC.ICEServers {
		server := webrtc.ICEServer{
			URLs:       iceServer.URLs,
			Username:   iceServer.Username,
			Credential: iceServer.Credential,
		}
		iceServers = append(iceServers, server)
	}

	routers = make(map[string]*Router)
	CleanRouter = make(chan string, maxCleanSize)

	// 启动Route清理线程
	go CheckRoute()
}

// 销毁RTC
func FreeRTC() {
	stop = true
	routersLock.Lock()
	defer routersLock.Unlock()
	for id, router := range routers {
		if router != nil {
			router.Close()
			delete(routers, id)
		}
	}
}

// GetOrNewRouter 获取router
func GetOrNewRouter(id string) *Router {
	router := GetRouter(id)
	if router == nil {
		return AddRouter(id)
	}
	return router
}

// GetRouters 获取所有Router
func GetRouters() map[string]*Router {
	return routers
}

// GetRouter 获取Router
func GetRouter(id string) *Router {
	routersLock.Lock()
	defer routersLock.Unlock()
	return routers[id]
}

// AddRouter 增加Router
func AddRouter(id string) *Router {
	logger.Debugf("add router = %s", id)
	router := NewRouter(id)
	routersLock.Lock()
	defer routersLock.Unlock()
	routers[id] = router
	return router
}

// DelRouter 删除Router
func DelRouter(id string) {
	router := GetRouter(id)
	if router != nil {
		logger.Debugf("del router = %s", id)
		router.Close()
		routersLock.Lock()
		defer routersLock.Unlock()
		delete(routers, id)
	}
}

// CheckRoute 查询所有router的状态
func CheckRoute() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for {
		if stop {
			return
		}

		<-t.C
		routersLock.Lock()
		for id, Router := range routers {
			if !Router.Alive() {
				logger.Debugf("router is dead = %s", id)
				Router.Close()
				delete(routers, id)
				CleanRouter <- id
			}
		}
		routersLock.Unlock()
	}
}
