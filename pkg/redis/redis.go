package redis

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// Config Redis配置对象
type Config struct {
	Addrs []string
	Pwd   string
	DB    int
}

// Redis Redis对象
type Redis struct {
	cluster     *redis.ClusterClient
	single      *redis.Client
	clusterMode bool
}

// NewRedis 创建Redis对象
func NewRedis(c Config) *Redis {
	if len(c.Addrs) == 0 {
		return nil
	}

	r := &Redis{}
	if len(c.Addrs) == 1 {
		// 单对象赋值
		r.clusterMode = false
		r.single = redis.NewClient(
			&redis.Options{
				Addr:         c.Addrs[0],
				Password:     c.Pwd,
				DB:           c.DB,
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		err := r.single.Ping(context.Background()).Err()
		if err != nil {
			log.Println(err.Error())
			return nil
		}
		return r
	} else {
		// 集群对象赋值
		r.clusterMode = true
		r.cluster = redis.NewClusterClient(
			&redis.ClusterOptions{
				Addrs:        c.Addrs,
				Password:     c.Pwd,
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		err := r.cluster.Ping(context.Background()).Err()
		if err != nil {
			log.Println(err.Error())
			return nil
		}
	}
	return r
}

// Keys redis查找key是否存在
func (r *Redis) Exists(k string) int64 {
	if r.clusterMode {
		return r.cluster.Exists(context.Background(), k).Val()
	}
	return r.single.Exists(context.Background(), k).Val()
}

// Keys redis查找所有符合给定模式的所有key
func (r *Redis) Keys(k string) []string {
	if r.clusterMode {
		return r.cluster.Keys(context.Background(), k).Val()
	}
	return r.single.Keys(context.Background(), k).Val()
}

// Del redis删除指定key的所有数据
func (r *Redis) Del(k string) error {
	if r.clusterMode {
		return r.cluster.Del(context.Background(), k).Err()
	}
	return r.single.Del(context.Background(), k).Err()
}

// Expire redis设置key过期时间
func (r *Redis) Expire(k string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Expire(context.Background(), k, t).Err()
	}
	return r.single.Expire(context.Background(), k, t).Err()
}

// Set redis以字符串方式存储key值
func (r *Redis) Set(k, v string, t time.Duration) error {
	if r.clusterMode {
		return r.cluster.Set(context.Background(), k, v, t).Err()
	}
	return r.single.Set(context.Background(), k, v, t).Err()
}

// SetNx 不存在则写入
func (r *Redis) SetNx(k, v string, t time.Duration) bool {
	if r.clusterMode {
		return r.cluster.SetNX(context.Background(), k, v, t).Val()
	}
	return r.single.SetNX(context.Background(), k, v, t).Val()
}

// Get redis以字符串方式存储,获取key值
func (r *Redis) Get(k string) string {
	if r.clusterMode {
		return r.cluster.Get(context.Background(), k).Val()
	}
	return r.single.Get(context.Background(), k).Val()
}

// HSet redis以hash散列表方式存储key的field字段的值
func (r *Redis) HSet(k, field string, value interface{}) error {
	if r.clusterMode {
		return r.cluster.HSet(context.Background(), k, field, value).Err()
	}
	return r.single.HSet(context.Background(), k, field, value).Err()
}

// HGet redis读取hash散列表key的field字段的值
func (r *Redis) HGet(k, field string) string {
	if r.clusterMode {
		return r.cluster.HGet(context.Background(), k, field).Val()
	}
	return r.single.HGet(context.Background(), k, field).Val()
}

// HMSet redis以hash散列表方式存储key的field字段的值
func (r *Redis) HMSet(k string, fields map[string]interface{}) error {
	if r.clusterMode {
		return r.cluster.HMSet(context.Background(), k, fields).Err()
	}
	return r.single.HMSet(context.Background(), k, fields).Err()
}

// HMGet redis读取hash散列表key的field字段的值
func (r *Redis) HMGet(k, field string) []interface{} {
	if r.clusterMode {
		return r.cluster.HMGet(context.Background(), k, field).Val()
	}
	return r.single.HMGet(context.Background(), k, field).Val()
}

// HDel redis删除hash散列表key的field字段
func (r *Redis) HDel(k, field string) error {
	if r.clusterMode {
		return r.cluster.HDel(context.Background(), k, field).Err()
	}
	return r.single.HDel(context.Background(), k, field).Err()
}

// HGetAll redis读取hash散列表key值对应的全部字段数据
func (r *Redis) HGetAll(k string) map[string]string {
	if r.clusterMode {
		return r.cluster.HGetAll(context.Background(), k).Val()
	}
	return r.single.HGetAll(context.Background(), k).Val()
}
