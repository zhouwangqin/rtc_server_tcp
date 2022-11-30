package src

import (
	"fmt"
	"server/pkg/proto"
	"server/pkg/util"
	"strings"

	"github.com/zhuanxin-sz/go-protoo/logger"
	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

// 处理rpc请求
func handleRpcMsg(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	go handleRPCRequest(request, accept, reject)
}

// 接收biz消息处理
func handleRPCRequest(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
	defer util.Recover("islb.handleRPCRequest")

	method := request["method"].(string)
	data := request["data"].(map[string]interface{})

	var result map[string]interface{}
	err := &nprotoo.Error{Code: 400, Reason: fmt.Sprintf("Unkown method [%s]", method)}

	/* 处理和其它服务器通信 */
	switch method {
	case proto.BizToIslbOnJoin:
		result, err = clientJoin(data)
	case proto.BizToIslbOnLeave:
		result, err = clientLeave(data)
	case proto.BizToIslbKeepAlive:
		result, err = keepalive(data)
	case proto.BizToIslbOnStreamAdd:
		result, err = streamAdd(data)
	case proto.BizToIslbOnStreamRemove:
		result, err = streamRemove(data)
	case proto.BizToIslbGetBizInfo:
		result, err = getBizByUid(data)
	case proto.BizToIslbGetSfuInfo:
		result, err = getSfuByMid(data)
	case proto.BizToIslbGetRoomUsers:
		result, err = getRoomUsers(data)
	case proto.BizToIslbGetRoomPubs:
		result, err = getRoomPubs(data)
	}
	// 判断成功
	if err != nil {
		reject(err.Code, err.Reason)
	} else {
		accept(result)
	}
}

/*
	"method", proto.BizToIslbOnJoin, "rid", rid, "uid", uid, "bizid", bizid
*/
// 有人加入房间
func clientJoin(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.clientJoin data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	bizid := util.Val(data, "bizid")
	// 获取用户的Biz服务器
	uKey := proto.GetUserNodeKey(rid, uid)
	err := redis.Set(uKey, bizid, redisShort)
	if err != nil {
		logger.Errorf("islb.clientJoin redis.Set err=%v, data=%v", err, data)
		return nil, &nprotoo.Error{Code: 401, Reason: fmt.Sprintf("clientJoin err=%v", err)}
	}
	return util.Map("rid", rid, "uid", uid, "bizid", bizid), nil
}

/*
	"method", proto.BizToIslbOnLeave, "rid", rid, "uid", uid
*/
// 有人退出房间
func clientLeave(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.clientLeave data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的Biz服务器
	uKey := proto.GetUserNodeKey(rid, uid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		err := redis.Del(uKey)
		if err != nil {
			logger.Errorf("islb.clientLeave redis.Del err=%v, data=%v", err, data)
		}
	}
	return util.Map("rid", rid, "uid", uid), nil
}

/*
	"method", proto.BizToIslbKeepAlive, "rid", rid, "uid", uid
*/
// 保活处理
func keepalive(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的Biz服务器
	uKey := proto.GetUserNodeKey(rid, uid)
	err := redis.Expire(uKey, redisShort)
	if err != nil {
		logger.Errorf("islb.keepalive redis.Expire err=%v, data=%v", err, data)
		return nil, &nprotoo.Error{Code: 402, Reason: fmt.Sprintf("keepalive err=%v", err)}
	}
	return util.Map("rid", rid, "uid", uid), nil
}

/*
	"method", proto.BizToIslbOnStreamAdd, "rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", minfo
*/
// 有人发布流
func streamAdd(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.streamAdd data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	sfuid := util.Val(data, "sfuid")
	minfo := util.Val(data, "minfo")
	// 获取用户流的信息
	ukey := proto.GetMediaInfoKey(rid, uid, mid)
	err := redis.Set(ukey, minfo, redisKeyTTL)
	if err != nil {
		logger.Errorf("islb.streamAdd media redis.Set err=%v, data=%v", err, data)
		return nil, &nprotoo.Error{Code: 405, Reason: fmt.Sprintf("streamAdd err=%v", err)}
	}
	// 获取用户流的sfu服务器
	ukey = proto.GetMediaPubKey(rid, uid, mid)
	err = redis.Set(ukey, sfuid, redisKeyTTL)
	if err != nil {
		logger.Errorf("islb.streamAdd pub redis.Set err=%v, data=%v", err, data)
		return nil, &nprotoo.Error{Code: 406, Reason: fmt.Sprintf("streamAdd err=%v", err)}
	}
	// 生成resp对象
	return util.Map("rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", data["minfo"]), nil
}

/*
	"method", proto.BizToIslbOnStreamRemove, "rid", rid, "uid", uid, "mid", ""
*/
// 有人取消发布流
func streamRemove(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.streamRemove data=%v", data)
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	mid := util.Val(data, "mid")
	// 判断mid是否为空
	rmPubs := make([]map[string]interface{}, 0)
	var ukey string
	if mid == "" {
		ukey = "/media/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys := redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf("islb.streamRemove media redis.Del err=%v, data=%v", err, data)
			}
		}
		ukey = "/pub/rid/" + rid + "/uid/" + uid + "/mid/*"
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			arr := strings.Split(key, "/")
			mid := arr[7]
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf("islb.streamRemove pub redis.Del err=%v, data=%v", err, data)
			}
			rmPubs = append(rmPubs, util.Map("rid", rid, "uid", uid, "mid", mid))
		}
	} else {
		// 获取用户流的信息
		ukey = proto.GetMediaInfoKey(rid, uid, mid)
		ukeys := redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf("islb.streamRemove media redis.Del err=%v, data=%v", err, data)
			}
		}
		// 获取用户流的sfu服务器
		ukey = proto.GetMediaPubKey(rid, uid, mid)
		ukeys = redis.Keys(ukey)
		for _, key := range ukeys {
			ukey = key
			arr := strings.Split(key, "/")
			mid := arr[7]
			// 删除key值
			err := redis.Del(ukey)
			if err != nil {
				logger.Errorf("islb.streamRemove pub redis.Del err=%v, data=%v", err, data)
			}
			rmPubs = append(rmPubs, util.Map("rid", rid, "uid", uid, "mid", mid))
		}
	}
	return util.Map("rmPubs", rmPubs), nil
}

/*
	"method", proto.BizToIslbGetBizInfo, "rid", rid, "uid", uid
*/
// 获取uid指定的biz节点信息
func getBizByUid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	uid := util.Val(data, "uid")
	// 获取用户的Biz服务器
	uKey := proto.GetUserNodeKey(rid, uid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		bizid := redis.Get(uKey)
		return util.Map("rid", rid, "uid", uid, "bizid", bizid), nil
	} else {
		return nil, &nprotoo.Error{Code: 410, Reason: fmt.Sprintf("can't find biz node by key:%s", uKey)}
	}
}

/*
	"method", proto.BizToIslbGetSfuInfo, "rid", rid, "mid", mid
*/
// 获取mid指定对应的sfu节点
func getSfuByMid(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	rid := util.Val(data, "rid")
	mid := util.Val(data, "mid")
	uid := proto.GetUIDFromMID(mid)
	// 获取用户流的sfu服务器
	uKey := proto.GetMediaPubKey(rid, uid, mid)
	ukeys := redis.Keys(uKey)
	if len(ukeys) > 0 {
		sfuid := redis.Get(uKey)
		return util.Map("rid", rid, "sfuid", sfuid), nil
	} else {
		return nil, &nprotoo.Error{Code: 411, Reason: fmt.Sprintf("can't find sfu node by key:%s", uKey)}
	}
}

/*
	"method", proto.BizToIslbGetRoomUsers, "rid", rid, "uid", uid
*/
// 获取房间其他用户数据
func getRoomUsers(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.getRoomUsers data=%v", data)
	rid := util.Val(data, "rid")
	id := util.Val(data, "uid")
	// 查询数据库
	users := make([]map[string]interface{}, 0)
	uKey := "/node/rid/" + rid + "/uid/*"
	ukeys := redis.Keys(uKey)
	for _, key := range ukeys {
		// 去掉指定的uid
		arr := strings.Split(key, "/")
		uid := arr[5]
		if uid == id {
			continue
		}

		bizid := redis.Get(key)
		user := util.Map("rid", rid, "uid", uid, "bizid", bizid)
		users = append(users, user)
	}
	// 返回
	resp := util.Map("users", users)
	return resp, nil
}

/*
	"method", proto.BizToIslbGetRoomPubs, "rid", rid, "uid", uid
*/
// 获取房间其他用户推流数据
func getRoomPubs(data map[string]interface{}) (map[string]interface{}, *nprotoo.Error) {
	logger.Debugf("islb.getRoomPubs data=%v", data)
	rid := util.Val(data, "rid")
	id := util.Val(data, "uid")
	// 查询数据库
	pubs := make([]map[string]interface{}, 0)
	uKey := "/pub/rid/" + rid + "/uid/*"
	ukeys := redis.Keys(uKey)
	for _, key := range ukeys {
		// 去掉指定的uid
		arr := strings.Split(key, "/")
		uid := arr[5]
		mid := arr[7]
		if uid == id {
			continue
		}

		sfuid := redis.Get(key)
		mKey := proto.GetMediaInfoKey(rid, uid, mid)
		minfo := redis.Get(mKey)

		pub := util.Map("rid", rid, "uid", uid, "mid", mid, "sfuid", sfuid, "minfo", util.Unmarshal(minfo))
		pubs = append(pubs, pub)
	}
	// 返回
	resp := util.Map("pubs", pubs)
	return resp, nil
}
