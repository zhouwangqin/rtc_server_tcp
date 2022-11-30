package proto

import (
	"strings"
)

const (
	/*
		客户端与biz服务器通信
	*/

	// ClientToBizJoin C->Biz 加入会议
	ClientToBizJoin = "join"
	// ClientToBizLeave C->Biz 离开会议
	ClientToBizLeave = "leave"
	// ClientToBizKeepAlive C->Biz 保活
	ClientToBizKeepAlive = "keepalive"
	// ClientToBizPublish C->Biz 发布流
	ClientToBizPublish = "publish"
	// ClientToBizUnPublish C->Biz 取消发布流
	ClientToBizUnPublish = "unpublish"
	// ClientToBizSubscribe C->Biz 订阅流
	ClientToBizSubscribe = "subscribe"
	// ClientToBizUnSubscribe C->Biz 取消订阅流
	ClientToBizUnSubscribe = "unsubscribe"
	// ClientToBizBroadcast C->Biz 发送广播
	ClientToBizBroadcast = "broadcast"
	// ClientToBizGetRoomUsers C->Biz 获取房间所有用户数据
	ClientToBizGetRoomUsers = "getusers"
	// ClientToBizGetRoomPubs C->Biz 获取房间所有用户流数据
	ClientToBizGetRoomPubs = "getpubs"

	// BizToClientOnJoin Biz->C 有人加入房间
	BizToClientOnJoin = "peer-join"
	// BizToClientOnLeave Biz->C 有人离开房间
	BizToClientOnLeave = "peer-leave"
	// BizToClientOnStreamAdd Biz->C 有人发布流
	BizToClientOnStreamAdd = "stream-add"
	// BizToClientOnStreamRemove Biz->C 有人取消发布流
	BizToClientOnStreamRemove = "stream-remove"
	// BizToClientBroadcast Biz->C 有人发送广播
	BizToClientBroadcast = "broadcast"
	// BizToClientOnKick Biz->C 被服务器踢下线
	BizToClientOnKick = "peer-kick"

	/*
		biz与biz服务器通信
	*/

	//BizToBizOnJoin biz->biz 有人加入房间
	BizToBizOnJoin = BizToClientOnJoin
	//BizToBizOnLeave biz->biz 有人离开房间
	BizToBizOnLeave = BizToClientOnLeave
	//BizToBizStreamAdd biz->biz 流添加
	BizToBizOnStreamAdd = BizToClientOnStreamAdd
	//BizToBizStreamRemove biz->biz 流移除
	BizToBizOnStreamRemove = BizToClientOnStreamRemove
	// BizToBizBroadcast biz->biz 有人发送广播
	BizToBizBroadcast = BizToClientBroadcast
	// BizToBizOnKick biz->biz 被服务器踢下线
	BizToBizOnKick = BizToClientOnKick

	/*
		biz与sfu服务器通信
	*/

	// BizToSfuPublish Biz->Sfu 发布流
	BizToSfuPublish = ClientToBizPublish
	// BizToSfuUnPublish Biz->Sfu 取消发布流
	BizToSfuUnPublish = ClientToBizUnPublish
	// BizToSfuSubscribe Biz->Sfu 订阅流
	BizToSfuSubscribe = ClientToBizSubscribe
	// BizToSfuUnSubscribe Biz->Sfu 取消订阅流
	BizToSfuUnSubscribe = ClientToBizUnSubscribe
	// SfuToBizOnStreamRemove Sfu->Biz Sfu通知biz流被移除
	SfuToBizOnStreamRemove = "sfu-stream-remove"

	/*
		biz与islb服务器通信
	*/

	// BizToIslbOnJoin biz->islb 有人加入房间
	BizToIslbOnJoin = "peer-join"
	// BizToIslbOnLeave biz->islb 有人离开房间
	BizToIslbOnLeave = "peer-leave"
	// BizToIslbKeepAlive biz->islb 有人保活
	BizToIslbKeepAlive = "keepalive"
	// BizToIslbOnStreamAdd biz->islb 有人开始推流
	BizToIslbOnStreamAdd = "stream-add"
	// BizToIslbOnStreamRemove biz->islb 有人停止推流
	BizToIslbOnStreamRemove = "stream-remove"
	// BizToIslbGetBizInfo biz->islb 根据uid查询对应的biz
	BizToIslbGetBizInfo = "getBizInfo"
	// BizToIslbGetSfuInfo biz->islb 根据mid查询对应的sfu
	BizToIslbGetSfuInfo = "getSfuInfo"
	// BizToIslbGetRoomUsers biz->islb 获取房间其他用户数据
	BizToIslbGetRoomUsers = "getRoomUsers"
	// BizToIslbGetRoomPubs biz->islb 获取房间其他用户推流数据
	BizToIslbGetRoomPubs = "getRoomPubs"
)

// GetUIDFromMID 从mid中获取uid
func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

// GetUserNodeKey 获取用户的Biz服务器
func GetUserNodeKey(rid, uid string) string {
	return "/node/rid/" + rid + "/uid/" + uid
}

// GetMediaInfoKey 获取用户流的信息
func GetMediaInfoKey(rid, uid, mid string) string {
	return "/media/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}

// GetMediaPubKey 获取用户流的sfu服务器
func GetMediaPubKey(rid, uid, mid string) string {
	return "/pub/rid/" + rid + "/uid/" + uid + "/mid/" + mid
}
