package src

import (
	"server/pkg/proto"
	"server/pkg/util"

	"github.com/zhuanxin-sz/go-protoo/logger"
	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

// handlerWebsocket 信令处理
func handlerWebsocket(method string, peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	switch method {
	case proto.ClientToBizJoin:
		join(peer, msg, accept, reject)
	case proto.ClientToBizLeave:
		leave(peer, msg, accept, reject)
	case proto.ClientToBizKeepAlive:
		keepalive(peer, msg, accept, reject)
	case proto.ClientToBizPublish:
		publish(peer, msg, accept, reject)
	case proto.ClientToBizUnPublish:
		unpublish(peer, msg, accept, reject)
	case proto.ClientToBizSubscribe:
		subscribe(peer, msg, accept, reject)
	case proto.ClientToBizUnSubscribe:
		unsubscribe(peer, msg, accept, reject)
	case proto.ClientToBizBroadcast:
		broadcast(peer, msg, accept, reject)
	case proto.ClientToBizGetRoomUsers:
		getusers(peer, msg, accept, reject)
	case proto.ClientToBizGetRoomPubs:
		getpubs(peer, msg, accept, reject)
	default:
		DefaultReject(codeUnknownErr, codeStr(codeUnknownErr))
	}
}

/*
  "request":true
  "id":3764139
  "method":"join"
  "data":{
    "rid":"room"
	"uid":"123456"
  }
*/
// 用户加入房间
func join(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	if invalid(msg, "uid", reject) {
		return
	}

	uid := util.Val(msg, "uid")
	rid := util.Val(msg, "rid")
	peer.id = uid

	// 获取islb服务器RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 查询uid是否在房间中
	// resp = "rid", rid, "uid", uid, "bizid", bizid
	resp, err := islbRpc.SyncRequest(proto.BizToIslbGetBizInfo, util.Map("rid", rid, "uid", uid))
	if err == nil {
		// uid已经存在，先删除
		bizid := resp["bizid"].(string)
		if bizid != node.NodeInfo().Nid {
			// 不在当前节点,通知其他节点关闭
			rpcBiz := rpcs[bizid]
			if rpcBiz != nil {
				rpcBiz.SyncRequest(proto.BizToBizOnKick, util.Map("rid", rid, "uid", uid))
			}
		} else {
			// 在当前节点

			// 删除数据库流
			// resp = "rmPubs", rmPubs
			// pub = "rid", rid, "uid", uid, "mid", mid
			resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
			if err == nil {
				rmPubs, ok := resp["rmPubs"].([]interface{})
				if ok {
					SendNotifysByUid(rid, uid, proto.BizToBizOnStreamRemove, rmPubs)
				}
			} else {
				logger.Errorf("biz.join request islb streamRemove err:%s", err.Reason)
			}

			// 删除数据库人
			// resp = util.Map("rid", rid, "uid", uid)
			_, err = islbRpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
			if err != nil {
				logger.Errorf("biz.join request islb clientLeave err:%s", err.Reason)
			}

			// 发送广播给所有人
			SendNotifyByUid(rid, uid, proto.BizToBizOnLeave, resp)
			// 通知本地对象
			//rooms.NotifyWithUid(rid, uid, proto.BizToClientOnKick, resp)
			// 删除本地对象
			room := rooms.GetRoom(rid)
			if room != nil {
				room.DelPeer(uid)
			}
		}
	}

	// 重新加入房间
	room := rooms.AddRoom(rid)
	room.AddPeer(peer)

	// 写数据库
	// resp = "rid", rid, "uid", uid, "bizid", bizid
	resp, err = islbRpc.SyncRequest(proto.BizToIslbOnJoin, util.Map("rid", rid, "uid", uid, "bizid", node.NodeInfo().Nid))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}

	// 广播通知房间其他人
	SendNotifyByUid(rid, uid, proto.BizToBizOnJoin, resp)

	_, users := FindRoomUsers(rid, uid)
	_, pubs := FindRoomPubs(rid, uid)
	result := util.Map("users", users, "pubs", pubs)
	accept(result)
}

/*
  "request":true
  "id":3764139
  "method":"leave"
  "data":{
      "rid":"room"
  }
*/
// leave 离开房间
func leave(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 删除数据库流
	// resp = "rmPubs", rmPubs
	resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", ""))
	if err == nil {
		rmPubs, ok := resp["rmPubs"].([]interface{})
		if ok {
			SendNotifysByUid(rid, uid, proto.BizToBizOnStreamRemove, rmPubs)
		}
	} else {
		logger.Errorf("biz.leave request islb streamRemove err:%s", err.Reason)
	}

	// 删除数据库人
	// resp = util.Map("rid", rid, "uid", uid)
	resp, err = islbRpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf("biz.leave request islb clientLeave err:%s", err.Reason)
	}

	// 发送广播给其他人
	SendNotifyByUid(rid, uid, proto.BizToClientOnLeave, resp)
	accept(emptyMap)
	// 删除本地
	room := rooms.GetRoom(rid)
	if room != nil {
		room.DelPeer(uid)
	}
}

/*
  "request":true
  "id":3764139
  "method":"keepalive"
  "data":{
    "rid":"room",
  }
*/
// keepalive 保活
func keepalive(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 判断是否在房间
	room := rooms.GetRoom(rid)
	if room == nil {
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 更新数据库
	// resp = "rid", rid, "uid", uid
	_, err := islbRpc.SyncRequest(proto.BizToIslbKeepAlive, util.Map("rid", rid, "uid", uid))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"publish"
  "data":{
      "rid":"room",
      "jsep": {"type": "offer","sdp": "..."},
      "minfo": {
	  	"audio": true,
	  	"video": true,
		"videotype": 0
	  }
  }
*/
// publish 发布流
func publish(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	minfo, ok := msg["minfo"].(map[string]interface{})
	if minfo == nil || !ok {
		reject(codeMinfoErr, codeStr(codeMinfoErr))
		return
	}

	// 判断是否在房间
	room := rooms.GetRoom(rid)
	if room == nil {
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 根据payload获取sfu RPC句柄
	sfuRpc, sfuid := GetRPCHandlerByPayload("sfu")
	if sfuRpc == nil {
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	// 获取sfu节点的resp
	// resp = "mid", mid, "jsep", util.Map("type", "answer", "sdp", resp)
	resp, err := sfuRpc.SyncRequest(proto.BizToSfuPublish, util.Map("rid", rid, "uid", uid, "jsep", jsep))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}

	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 写数据库流
	// resp = "rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", data["minfo"]
	mid := util.Val(resp, "mid")
	stream, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamAdd, util.Map("rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", minfo))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}

	// 发广播给其他人
	SendNotifyByUid(rid, uid, proto.BizToBizOnStreamAdd, stream)

	// resp
	rsp := make(map[string]interface{})
	rsp["mid"] = mid
	rsp["sfuid"] = sfuid
	rsp["jsep"] = resp["jsep"]
	accept(rsp)
}

/*
  "request":true
  "id":3764139
  "method":"unpublish"
  "data":{
      "rid": "room",
      "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
  	  "sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// unpublish 取消发布流
func unpublish(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	// 获取sfu RPC句柄
	var sfuRpc *nprotoo.Requestor
	sfuid := util.Val(msg, "sfuid")
	if sfuid != "" {
		sfuRpc = GetRPCHandlerByNodeID(sfuid)
	} else {
		sfuRpc = GetSFURPCHandlerByMID(rid, mid)
	}
	if sfuRpc == nil {
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	// 获取sfu节点的resp
	// resp = util.Map()
	_, err := sfuRpc.SyncRequest(proto.BizToSfuUnPublish, util.Map("rid", rid, "mid", mid))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}

	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
		return
	}

	// 删除数据库流
	// resp =  util.Map("rmPubs", rmPubs)
	resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}

	// 发送广播给其他人
	rmPubs, ok := resp["rmPubs"].([]interface{})
	if ok {
		SendNotifysByUid(rid, uid, proto.BizToClientOnStreamRemove, rmPubs)
	}
	accept(emptyMap)
}

/*
  "request":true
  "id":3764139
  "method":"subscribe"
  "data":{
    "rid":"room",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
	"jsep": {"type": "offer","sdp": "..."},
	"sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// subscribe 订阅流
func subscribe(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "jsep", reject) {
		return
	}

	jsep := msg["jsep"].(map[string]interface{})
	if invalid(jsep, "sdp", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")

	// 判断是否在房间
	room := rooms.GetRoom(rid)
	if room == nil {
		reject(codeRIDErr, codeStr(codeRIDErr))
		return
	}

	// 获取sfu RPC句柄
	var sfuRpc *nprotoo.Requestor
	sfuid := util.Val(msg, "sfuid")
	if sfuid != "" {
		sfuRpc = GetRPCHandlerByNodeID(sfuid)
	} else {
		sfuRpc = GetSFURPCHandlerByMID(rid, mid)
	}
	if sfuRpc == nil {
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	// 获取sfu节点的resp
	// resp = "sid", sid, "jsep", util.Map("type", "answer", "sdp", resp)
	resp, err := sfuRpc.SyncRequest(proto.BizToSfuSubscribe, util.Map("rid", rid, "suid", uid, "mid", mid, "jsep", jsep))
	if err != nil {
		// 流已经不存在了
		if err.Code == 403 {
			// 获取islb RPC句柄
			islbRpc := GetRPCHandlerByServiceName("islb")
			if islbRpc == nil {
				reject(codeIslbRpcErr, codeStr(codeIslbRpcErr))
				return
			}

			// 删除数据库流
			// resp =  util.Map("rmPubs", rmPubs)
			id := proto.GetUIDFromMID(mid)
			resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", id, "mid", mid))
			if err != nil {
				reject(err.Code, err.Reason)
				return
			}

			// 发送广播给其他人
			rmPubs, ok := resp["rmPubs"].([]interface{})
			if ok {
				SendNotifysByUid(rid, id, proto.BizToClientOnStreamRemove, rmPubs)
			}
		}
		reject(err.Code, err.Reason)
	} else {
		accept(resp)
	}
}

/*
  "request":true
  "id":3764139
  "method":"unsubscribe"
  "data":{
    "rid": "room",
    "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
    "sid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF"
	"sfuid":"shenzhen-sfu-1", (可选)
  }
*/
// unsubscribe 取消订阅流
func unsubscribe(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) || invalid(msg, "mid", reject) || invalid(msg, "sid", reject) {
		return
	}

	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	sid := util.Val(msg, "sid")

	// 获取sfu RPC句柄
	var sfuRpc *nprotoo.Requestor
	sfuid := util.Val(msg, "sfuid")
	if sfuid != "" {
		sfuRpc = GetRPCHandlerByNodeID(sfuid)
	} else {
		sfuRpc = GetSFURPCHandlerByMID(rid, mid)
	}
	if sfuRpc == nil {
		reject(codeSfuRpcErr, codeStr(codeSfuRpcErr))
		return
	}

	// 获取sfu节点的resp
	// resp = util.Map()
	_, err := sfuRpc.SyncRequest(proto.BizToSfuUnSubscribe, util.Map("rid", rid, "mid", mid, "sid", sid))
	if err != nil {
		reject(err.Code, err.Reason)
		return
	}
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"broadcast"
	"data":{
		"rid": "room",
		"data": "$date"
	}
*/
// broadcast 客户端发送广播给对方
func broadcast(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")
	data := util.Map("rid", rid, "uid", uid, "data", msg["data"])
	// 发送广播
	SendNotifyByUid(rid, uid, proto.BizToClientBroadcast, data)
	accept(emptyMap)
}

/*
	"request":true
	"id":3764139
	"method":"getusers"
	"data":{
		"rid": "room",
	}
*/
// 获取房间其他用户数据
func getusers(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 查询房间其他用户数据
	_, users := FindRoomUsers(rid, uid)
	result := util.Map("users", users)
	accept(result)
}

/*
	"request":true
	"id":3764139
	"method":"getpubs"
	"data":{
		"rid": "room",
	}
*/
// 获取房间其他用户流数据
func getpubs(peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc) {
	if invalid(msg, "rid", reject) {
		return
	}

	uid := peer.ID()
	rid := util.Val(msg, "rid")

	// 查询房间其他用户流数据
	_, pubs := FindRoomPubs(rid, uid)
	result := util.Map("pubs", pubs)
	accept(result)
}
