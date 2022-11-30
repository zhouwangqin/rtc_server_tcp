package src

import (
	"net/http"
	_ "net/http/pprof"
	"server/pkg/etcd"
	db "server/pkg/redis"
	"server/server/islb/conf"
	"time"

	"github.com/zhuanxin-sz/go-protoo/logger"
	nprotoo "github.com/zhuanxin-sz/nats-protoo"
)

const (
	redisShort  = 60 * time.Second
	redisKeyTTL = 24 * time.Hour
)

var (
	redis *db.Redis
	node  *etcd.ServiceNode
	nats  *nprotoo.NatsProtoo
)

// Start 启动服务
func Start() {
	// 服务注册
	node = etcd.NewServiceNode(conf.Etcd.Addrs, conf.Global.Ndc, conf.Global.Nid, conf.Global.Name)
	node.RegisterNode()
	// 消息注册
	nats = nprotoo.NewNatsProtoo(conf.Nats.URL)
	nats.OnRequest(node.GetRPCChannel(), handleRpcMsg)
	// 数据库
	redis = db.NewRedis(db.Config(*conf.Redis))
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
}

func debug() {
	logger.Debugf("Start islb pprof on %s", conf.Global.Pprof)
	http.ListenAndServe(conf.Global.Pprof, nil)
}
