# 客户端与信令服务器通信协议

协议格式采用[protoo](https://protoo.versatica.com/)格式设计
协议使用Tcp连接，数据使用json格式

C = 客户端
S = 服务器
uid = 用户id
rid = 房间id
mid = 用户发布的流id
sid = 用户订阅流的id
bizid = 用户所在biz服务器id
sfuid = 用户流所在sfu服务器id

## Tcp连接
c-->s
$host:$port

## TCP 包格式，LV格式
L = 2个字节，包长度
V = json数据

## 加入房间
c-->s
{
	"request":true,
	"id":1928012,
	"method":"join",
	"data":{
		"rid":"777777"
		"uid":"111111"
	}
}

s-->c
// ok
{
	"response":true,
	"id":1928012,
	"ok":true,
	"data":{
		"pubs":[
			{
				"mid":"HUAWEI_94bf#661522",
				"minfo":{
					"audio":true,
					"audiotype":0,
					"video":false,
					"videotype":0
				},
				"rid":"100",
				"sfuid":"shenzhen_sfu_1",
				"uid":"HUAWEI_94bf"
			}
		],
		"users":[
			{
				"bizid":"shenzhen_biz_1",
				"rid":"100",
				"uid":"HUAWEI_94bf"
			}
		]
	}
}
// fail
{
	"response":true,
	"id":1928012,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 离开房间
c-->s
{
	"request":true,
	"id":3786561,
	"method":"leave",
	"data":{
		"rid":"777777"
	}
}

s-->c
// ok
{
	"response":true,
	"id":3786561,
	"ok":true,
	"data":{}
}
// fail
{
	"response":true,
	"id":3786561,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 房间内发心跳
c-->s
{
	"request":true,
	"id":6797364,
	"method":"keepalive",
	"data":{
		"rid":"777777"
	}
}
s-->c
// ok
{
	"response":true,
	"id":6797364,
	"ok":true,
	"data":{}
}
// fail
{
	"response":true,
	"id":6797364,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 发布流
c-->s
{
	"request":true
	"id":3764139
	"method":"publish"
	"data":{
		"rid":"room",
		"jsep":{"type":"offer","sdp":"..."},
		"minfo":{
			"audio":true,
			"video":true,
			"audiotype":0,
			"videotype":0,
		}
	}
}
s-->c
// ok
{
	"response":true,
	"id":3764139,
	"ok":true,
	"data":{
		"jsep":{
			"sdp":"$sdp",
			"type":"answer"
		},
		"mid":"samsung_10b8d#128047",
		"sfuid":"shenzhen_sfu_1"
	}
}
// fail
{
	"response":true,
	"id":3764139,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 取消发布流
c-->s
{
	"request":true,
	"id":3400298,
	"method":"unpublish",
	"data":{
		"rid":"777777",
		"mid":"samsung_10b8d#128047",
		"sfuid":"shenzhen_sfu_1"
	}
}
s-->c
// ok
{
	"response":true,
	"id":6797364,
	"ok":true,
	"data":{}
}
// fail
{
	"response":true,
	"id":6797364,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 订阅流
c-->s
{
	"request":true
	"id":3764139
	"method":"subscribe"
	"data":{
		"rid":"room",
		"mid":"64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
		"jsep":{
			"type":"offer",
			"sdp":"#sdp"
		},
		"sfuid":"shenzhen-sfu-1", (可选)
	}
}
s-->c
// ok
{
	"response":true,
	"id":3764139,
	"ok":true,
	"data":{
		"jsep":{
			"sdp":"$sdp",
			"type":"answer"
		},
		"sid":"samsung_1846e#678832"
	}
}
// fail
{
	"response":true,
	"id":3764139,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 取消订阅流
c-->s
{
	"request":true
    "id":3764139
    "method":"unsubscribe"
    "data":{
		"rid": "room",
        "mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
        "sid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
        "sfuid":"shenzhen-sfu-1", (可选)
    }
}
s-->c
// ok
{
	"response":true,
	"id":3764139,
	"ok":true,
	"data":{}
}
// fail
{
	"response":true,
	"id":3764139,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

## 发送广播
c-->s
{
	"request":true
	"id":3764139
	"method":"broadcast"
	"data":{
		"rid": "room",
		"data": "$date"
	}
}
s-->c
// ok
{
	"response":true,
	"id":3764139,
	"ok":true,
	"data":{}
}
// fail
{
	"response":true,
	"id":3764139,
	"ok":false,
	"errorCode": $err,
	"errorReason": "$reason"
}

/* 
	服务器主动通知 s-->c
*/

## 有人加入房间
{
	"notification" : true,
	"method": "peer-join",
	"data":{
		"rid": "777777",
		"uid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f",
		"bizid", bizid
	}
}

## 有人离开房间
{
	"notification" : true,
	"method": "peer-leave",
	"data":{
		"rid": "777777"
		"uid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f",
	}
}

## 有人发布流
{
	"notification" : true,
	"method":"stream-add",
	"data":{
		"rid": "777777",
		"uid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f",
		"mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
		"sfuid":"shenzhen-sfu-1",
		"minfo": {
			"audio":true,
			"video":true,
			"audiotype":0,
			"videotype":0,
		}
	}
}

## 有人取消发布流
{
	"notification" : true,
	"method":"stream-remove",
	"data":{
		"rid": "777777",
		"uid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f",
		"mid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f#ABCDEF",
	}
}

## 有人发广播
{
	"notification" : true,
	"method":"broadcast",
	"data":{
		"rid": "777777",
		"uid": "64236c21-21e8-4a3d-9f80-c767d1e1d67f",
		"data": "$date"
	}
}

