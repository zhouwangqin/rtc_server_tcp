package src

import (
	"fmt"
	"server/pkg/proto"
	"server/pkg/util"
	"server/server/sfu/rtc"

	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

// 处理sfu的rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

// 处理sfu的rpc请求
func handleRPCRequest(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	defer util.Recover("sfu.handleRPCRequest")

	method := request["method"].(string)
	data := request["data"].(map[string]interface{})
	var result map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

	if method != "" {
		switch method {
		case proto.BizToSfuPublish:
			result, err = publish(data)
		case proto.BizToSfuUnPublish:
			result, err = unpublish(data)
		case proto.BizToSfuSubscribe:
			result, err = subscribe(data)
		case proto.BizToSfuUnSubscribe:
			result, err = unsubscribe(data)
		}
	}
	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}

/*
	"method", proto.BizToSfuPublish, "rid", rid, "uid", uid, "jsep", jsep
*/
// publish 处理推流
func publish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	//logger.Debugf("sfu.publish msg=%v", msg)
	// 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: 401, Reason: "can't find jsep"}
	}
	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{Code: 402, Reason: "jsep can't transform to map"}
	}

	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	uid := util.Val(msg, "uid")
	mid := fmt.Sprintf("%s#%s", uid, util.RandStr(6))

	// 获取router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetOrNewRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{Code: 403, Reason: fmt.Sprintf("can't get router:%s", key)}
	}

	// 增加推流
	resp, err := router.AddPub(mid, sdp)
	if err != nil {
		return nil, &nprotoo.Error{Code: 404, Reason: fmt.Sprintf("add pub err:%v", err)}
	}
	return util.Map("mid", mid, "jsep", util.Map("type", "answer", "sdp", resp)), nil
}

/*
	"method", proto.BizToSfuUnPublish, "rid", rid, "mid", mid
*/
// unpublish 处理取消发布流
func unpublish(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	//logger.Debugf("sfu.unpublish msg=%v", msg)
	// 获取参数
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)

	// 删除router
	key := proto.GetMediaPubKey(rid, uid, mid)
	rtc.DelRouter(key)
	return util.Map(), nil
}

/*
	"method", proto.BizToSfuSubscribe, "rid", rid, "suid", suid, "mid", mid, "jsep", jsep
*/
// subscribe 处理订阅流
func subscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	//logger.Debugf("sfu.subscribe msg=%v", msg)
	// 获取参数
	if msg["jsep"] == nil {
		return nil, &nprotoo.Error{Code: 401, Reason: "can't find jsep"}
	}
	jsep, ok := msg["jsep"].(map[string]interface{})
	if !ok {
		return nil, &nprotoo.Error{Code: 402, Reason: "jsep can't transform to map"}
	}

	sdp := util.Val(jsep, "sdp")
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	uid := proto.GetUIDFromMID(mid)

	suid := util.Val(msg, "suid")
	sid := fmt.Sprintf("%s#%s", suid, util.RandStr(6))

	// 获取router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{Code: 403, Reason: fmt.Sprintf("can't get router:%s", key)}
	}

	// 增加拉流
	resp, err := router.AddSub(sid, sdp)
	if err != nil {
		return nil, &nprotoo.Error{Code: 404, Reason: fmt.Sprintf("add sub err:%v", err)}
	}
	return util.Map("sid", sid, "jsep", util.Map("type", "answer", "sdp", resp)), nil
}

/*
	"method", proto.BizToSfuUnSubscribe, "rid", rid, "mid", mid, "sid", sid
*/
// unsubscribe 处理取消订阅流
func unsubscribe(msg map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	//logger.Debugf("sfu.unsubscribe msg=%v", msg)
	// 获取参数
	rid := util.Val(msg, "rid")
	mid := util.Val(msg, "mid")
	sid := util.Val(msg, "sid")
	uid := proto.GetUIDFromMID(mid)

	// 获取router
	key := proto.GetMediaPubKey(rid, uid, mid)
	router := rtc.GetRouter(key)
	if router == nil {
		return nil, &nprotoo.Error{Code: 410, Reason: fmt.Sprintf("can't get router:%s", key)}
	}

	// 删除拉流
	router.DelSub(sid)
	return util.Map(), nil
}
