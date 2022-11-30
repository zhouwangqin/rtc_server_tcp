package src

import (
	"fmt"
	"log"
	"server/pkg/proto"
	"server/pkg/util"

	"github.com/zhuanxin-sz/go-protoo/logger"
	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

// 处理biz的rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

// 处理biz的rpc请求
func handleRPCRequest(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	//defer util.Recover("biz.handleRPCRequest")
	log.Printf("biz.handleRPCRequest request=%v", request)

	method := request["method"].(string)
	data := request["data"].(map[string]interface{})
	var result map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

	switch method {
	/* 处理和biz服务器通信 */
	case proto.BizToBizOnKick:
		result, err = peerKick(data)
	}
	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}

/*
	"method", proto.BizToBizOnKick, "rid", rid, "uid", uid
*/
// 踢出房间
func peerKick(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		return nil, &nprotoo.Error{Code: -1, Reason: "can't get available islb node"}
	}

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
		logger.Errorf("biz.peerKick request islb streamRemove err:%s", err.Reason)
	}

	// 删除数据库人
	// resp = util.Map("rid", rid, "uid", uid)
	_, err = islbRpc.SyncRequest(proto.BizToIslbOnLeave, util.Map("rid", rid, "uid", uid))
	if err != nil {
		logger.Errorf("biz.peerKick request islb clientLeave err:%s", err.Reason)
	}
	// 发送广播给所有人
	SendNotifyByUid(rid, uid, proto.BizToBizOnLeave, resp)

	// 通知客户端
	//NotifyPeerWithID(rid, uid, proto.BizToClientOnKick, util.Map("rid", rid, "uid", uid))

	// 删除本地对象
	room := rooms.GetRoom(rid)
	if room != nil {
		room.DelPeer(uid)
	}
	return util.Map(), nil
}

// handleBroadCastMsgs 处理广播消息
func handleBroadcast(msg map[string]interface{}, subj string) {
	defer util.Recover("biz.handleBroadcast")
	logger.Debugf("biz.handleBroadcast msg=%v", msg)

	method := util.Val(msg, "method")
	data := msg["data"].(map[string]interface{})
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")

	switch method {
	case proto.BizToBizOnJoin:
		/* "method", proto.BizToBizOnJoin, "rid", rid, "uid", uid, "bizid", bizid */
		NotifyPeersWithoutID(rid, uid, proto.BizToClientOnJoin, data)
	case proto.BizToBizOnLeave:
		/* "method", proto.BizToBizOnLeave, "rid", rid, "uid", uid */
		NotifyPeersWithoutID(rid, uid, proto.BizToClientOnLeave, data)
	case proto.BizToBizOnStreamAdd:
		/* "method", proto.BizToBizOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", data["minfo"] */
		NotifyPeersWithoutID(rid, uid, proto.BizToClientOnStreamAdd, data)
	case proto.BizToBizOnStreamRemove:
		/* "method", proto.BizToBizOnStreamRemove, "rid", rid, "uid", uid, "mid", mid */
		NotifyPeersWithoutID(rid, uid, proto.BizToClientOnStreamRemove, data)
	case proto.BizToBizBroadcast:
		/* "method", proto.BizToBizBroadcast, "rid", rid, "uid", uid, "data", data */
		NotifyPeersWithoutID(rid, uid, proto.BizToClientBroadcast, data)
	case proto.SfuToBizOnStreamRemove:
		mid := util.Val(data, "mid")
		sfuRemoveStream(rid, uid, mid)
	}
}

// 处理sfu移除流
func sfuRemoveStream(rid, uid, mid string) {
	// 获取islb RPC句柄
	islbRpc := GetRPCHandlerByServiceName("islb")
	if islbRpc == nil {
		return
	}

	resp, err := islbRpc.SyncRequest(proto.BizToIslbOnStreamRemove, util.Map("rid", rid, "uid", uid, "mid", mid))
	if err != nil {
		return
	}

	rmPubs, ok := resp["rmPubs"].([]interface{})
	if ok {
		SendNotifysByUid(rid, uid, proto.BizToClientOnStreamRemove, rmPubs)
	} else {
		return
	}
}

// NotifyPeersWithoutID 通知房间其他人
func NotifyPeersWithoutID(rid, uid, method string, msg map[string]interface{}) {
	rooms.NotifyWithoutUid(rid, uid, method, msg)
}

// NotifyPeerWithID 通知房间指定人
func NotifyPeerWithID(rid, uid, method string, msg map[string]interface{}) {
	rooms.NotifyWithUid(rid, uid, method, msg)
}
