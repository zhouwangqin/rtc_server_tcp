package src

import (
	"net/http"
	_ "net/http/pprof"
	"server/pkg/etcd"
	"server/pkg/proto"
	"server/pkg/util"
	"server/server/biz/conf"
	"time"

	"github.com/zhuanxin-sz/go-protoo/logger"
	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

const (
	statCycle = 10 * time.Second
)

var (
	rooms  *Rooms
	node   *etcd.ServiceNode
	watch  *etcd.ServiceWatcher
	nats   *nprotoo.NatsProtoo
	caster *nprotoo.Broadcaster
	rpcs   = make(map[string]*nprotoo.Requestor)
)

// Start 启动服务
func Start() {
	rooms = NewRooms()
	// 服务注册
	node = etcd.NewServiceNode(conf.Etcd.Addrs, conf.Global.Ndc, conf.Global.Nid, conf.Global.Name)
	node.RegisterNode()
	// 服务发现
	watch = etcd.NewServiceWatcher(conf.Etcd.Addrs)
	go watch.WatchServiceNode("", WatchServiceCallBack)
	// 消息注册
	nats = nprotoo.NewNatsProtoo(conf.Nats.URL)
	nats.OnRequest(node.GetRPCChannel(), handleRpcMsg)
	// 消息广播
	caster = nats.NewBroadcaster(node.GetEventChannel())
	// 启动tcp server
	go StartTcp(conf.Signal.Host, uint16(conf.Signal.Port))
	// 启动房间资源回收
	go CheckRoom()
	// 启动调试
	if conf.Global.Pprof != "" {
		go debug()
	}
}

// Stop 关闭服务
func Stop() {
	if nats != nil {
		nats.Close()
	}
	if node != nil {
		node.Close()
	}
	if watch != nil {
		watch.Close()
	}
}

func debug() {
	logger.Debugf("Start biz pprof on %s", conf.Global.Pprof)
	http.ListenAndServe(conf.Global.Pprof, nil)
}

// WatchServiceCallBack 查看所有的Node节点
func WatchServiceCallBack(state int32, n etcd.Node) {
	if state == etcd.ServerUp {
		// 判断是否广播节点
		if n.Name == "biz" {
			if n.Nid != node.NodeInfo().Nid {
				eventID := etcd.GetEventChannel(n)
				nats.OnBroadcast(eventID, handleBroadcast)
			}
		}
		if n.Name == "sfu" {
			eventID := etcd.GetEventChannel(n)
			nats.OnBroadcastWithGroup(eventID, "biz", handleBroadcast)
		}
		id := n.Nid
		_, found := rpcs[id]
		if !found {
			rpcID := etcd.GetRPCChannel(n)
			rpcs[id] = nats.NewRequestor(rpcID)
		}
	} else if state == etcd.ServerDown {
		delete(rpcs, n.Nid)
	}
}

// GetRPCHandlerByServiceName 通过服务名获取RPC Handler
func GetRPCHandlerByServiceName(name string) *nprotoo.Requestor {
	var tmp etcd.Node
	var node *etcd.Node
	services, find := watch.GetNodes(name)
	if find {
		for _, server := range services {
			tmp = server
			node = &tmp
			break
		}
	}
	if node != nil {
		rpc, find := rpcs[node.Nid]
		if find {
			return rpc
		}
	}
	return nil
}

// GetRPCHandlerByNodeID 获取指定id的获取RPC Handler
func GetRPCHandlerByNodeID(nid string) *nprotoo.Requestor {
	node, find := watch.GetNodeByID(nid)
	if !find {
		return nil
	}
	if node != nil {
		rpc, find := rpcs[node.Nid]
		if find {
			return rpc
		}
	}
	return nil
}

// GetRPCHandlerByPayload 获取最低负载的RPC Handler和id
func GetRPCHandlerByPayload(name string) (*nprotoo.Requestor, string) {
	node, find := watch.GetNodeByPayload(node.NodeInfo().Ndc, name)
	if !find {
		return nil, ""
	}
	rpc, find := rpcs[node.Nid]
	if find {
		return rpc, node.Nid
	}
	return nil, ""
}

// GetBizExistByUID 根据rid, uid判断人是否在线
func GetBizExistByUID(rid, uid string) bool {
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		logger.Errorf("GetBizExistByUID can't get available islb node")
		return false
	}

	// resp = "rid", rid, "uid", uid, "bizid", bizid
	resp, err := islbRpc.SyncRequest(proto.BizToIslbGetBizInfo, util.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false
	}

	//logger.Debugf("GetBizExistByUID resp ==> %v", resp)

	bizid := util.Val(resp, "bizid")
	if bizid != "" {
		if bizid == node.NodeInfo().Nid {
			return true
		} else {
			biz := GetRPCHandlerByNodeID(bizid)
			return (biz != nil)
		}
	}
	return false
}

// GetSFURPCHandlerByMID 根据rid, mid获取sfu节点rpc句柄
func GetSFURPCHandlerByMID(rid, mid string) *nprotoo.Requestor {
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		logger.Errorf("GetSFURPCHandlerByMID can't get available islb node")
		return nil
	}

	// resp = "rid", rid, "sfuid", sfuid
	resp, err := islbRpc.SyncRequest(proto.BizToIslbGetSfuInfo, util.Map("rid", rid, "mid", mid))
	if err != nil {
		logger.Errorf(err.Reason)
		return nil
	}

	logger.Infof("GetSFURPCHandlerByMID resp ==> %v", resp)

	var sfu *nprotoo.Requestor
	sfuid := util.Val(resp, "sfuid")
	if sfuid != "" {
		sfu = GetRPCHandlerByNodeID(sfuid)
	}
	return sfu
}

// FindRoomUsers 获取房间其他用户信息
func FindRoomUsers(rid, uid string) (bool, []interface{}) {
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		logger.Errorf("FindRoomUsers can't get available islb node")
		return false, nil
	}

	// resp = "users", users
	// user = "rid", rid, "uid", uid, "bizid", bizid
	resp, err := islbRpc.SyncRequest(proto.BizToIslbGetRoomUsers, util.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false, nil
	}

	logger.Infof("FindRoomUsers resp ==> %v", resp)

	if resp["users"] == nil {
		logger.Errorf("FindRoomUsers users is nil")
		return false, nil
	}

	users := resp["users"].([]interface{})
	return true, users
}

// FindRoomPubs 获取房间其他用户流信息
func FindRoomPubs(rid, uid string) (bool, []interface{}) {
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		logger.Errorf("FindRoomPubs can't get available islb node")
		return false, nil
	}

	// resp = "pubs", pubs
	// pub = "rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", util.Unmarshal(minfo)
	resp, err := islbRpc.SyncRequest(proto.BizToIslbGetRoomPubs, util.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf(err.Reason)
		return false, nil
	}

	logger.Infof("FindRoomPubs resp ==> %v", resp)

	if resp["pubs"] == nil {
		logger.Errorf("FindRoomPubs pubs is nil")
		return false, nil
	}

	pubs := resp["pubs"].([]interface{})
	return true, pubs
}

// CheckRoom 检查所有的房间
func CheckRoom() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		for rid, room := range rooms.GetRooms() {
			for uid := range room.GetPeers() {
				exist := GetBizExistByUID(rid, uid)
				if !exist {
					// 获取islb RPC句柄
					islbRpc := GetRPCHandlerByServiceName("islb")
					if islbRpc == nil {
						continue
					}
					// 删除数据库流
					// resp = "rmPubs", rmPubs
					// pub = "rid", rid, "uid", uid, "mid", mid
					resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
					if err == nil {
						rmPubs, ok := resp["rmPubs"].([]interface{})
						if ok {
							SendNotifysByUid(rid, uid, proto.BizToClientOnStreamRemove, rmPubs)
						}
					} else {
						logger.Errorf("biz.checkRoom request islb streamRemove err:%s", err.Reason)
					}
					// 删除数据库人
					// resp = "rid", rid, "uid", uid
					resp, err = islbRpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
					if err == nil {
						SendNotifyByUid(rid, uid, proto.BizToClientOnLeave, resp)
					} else {
						logger.Errorf("biz.checkRoom request islb clientLeave err:%s", err.Reason)
					}
					// 删除本地对象
					room.DelPeer(uid)
					logger.Debugf("room=%s del peer uid=%", rid, uid)
				}
			}
			if len(room.GetPeers()) == 0 {
				logger.Debugf("no peer in room=%s now", rid)
				rooms.DelRoom(rid)
			}
		}
	}
}

// SendNotifyByUid 单发广播给其他人
func SendNotifyByUid(rid, skipUid, method string, msg map[string]interface{}) {
	NotifyPeersWithoutID(rid, skipUid, method, msg)
	caster.Say(method, msg)
}

// SendNotifysByUid 群发广播给其他人
func SendNotifysByUid(rid, skipUid, method string, msgs []interface{}) {
	for _, msg := range msgs {
		data, ok := msg.(map[string]interface{})
		if ok {
			SendNotifyByUid(rid, skipUid, method, data)
		}
	}
}
