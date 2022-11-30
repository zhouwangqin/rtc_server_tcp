package etcd

import (
	"log"
	"strconv"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// ServerUp 服务存活
	ServerUp int32 = 0
	// ServerDown 服务死亡
	ServerDown int32 = 1
)

// ServiceWatchCallback 定义服务节点状态改变回调
type ServiceWatchCallback func(state int32, node Node)

// ServiceWatcher 服务发现对象
type ServiceWatcher struct {
	etcd     *Etcd
	bStop    bool
	nodes    map[string]Node
	nodeLook sync.Mutex
	callback ServiceWatchCallback
}

// NewServiceWatcher 新建一个服务发现对象
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	etcd, err := NewEtcd(endpoints)
	if err != nil {
		log.Printf("NewServiceWatcher err = %v", err)
		return nil
	}

	var serviceWatcher ServiceWatcher
	serviceWatcher.etcd = etcd
	serviceWatcher.bStop = false
	serviceWatcher.nodes = make(map[string]Node)
	return &serviceWatcher
}

// Close 关闭资源
func (serviceWatcher *ServiceWatcher) Close() {
	serviceWatcher.bStop = true
	if serviceWatcher.etcd != nil {
		serviceWatcher.etcd.Close()
	}
}

// GetNodes 根据服务名称获取所有该服务节点的所有对象
func (serviceWatcher *ServiceWatcher) GetNodes(serviceName string) (map[string]Node, bool) {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	mapNodes := make(map[string]Node)
	for _, node := range serviceWatcher.nodes {
		if node.Name == serviceName {
			mapNodes[node.Nid] = node
		}
	}
	if len(mapNodes) > 0 {
		return mapNodes, true
	}
	return mapNodes, false
}

// GetNodeByID 根据服务节点id获取到服务节点对象
func (serviceWatcher *ServiceWatcher) GetNodeByID(nid string) (*Node, bool) {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	node, find := serviceWatcher.nodes[nid]
	if find {
		return &node, true
	}
	return nil, false
}

// GetNodeByPayload 获取指定区域内指定服务节点负载最低的节点
func (serviceWatcher *ServiceWatcher) GetNodeByPayload(dc, name string) (*Node, bool) {
	var nodeTmp Node
	var nodePtr *Node
	var payload int = 65535
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	for _, node := range serviceWatcher.nodes {
		if node.Ndc == dc && node.Name == name {
			pay, _ := strconv.Atoi(node.Npay)
			if pay <= payload {
				nodeTmp = node
				nodePtr = &nodeTmp
				payload = pay
			}
		}
	}
	if nodePtr == nil {
		return nil, false
	}
	return nodePtr, true
}

// DeleteNodesByID 删除指定节点id的服务节点
func (serviceWatcher *ServiceWatcher) DeleteNodesByID(nid string) bool {
	serviceWatcher.nodeLook.Lock()
	defer serviceWatcher.nodeLook.Unlock()
	_, find := serviceWatcher.nodes[nid]
	if find {
		delete(serviceWatcher.nodes, nid)
	}
	return true
}

// WatchNode 监控到服务节点状态改变
func (serviceWatcher *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			if serviceWatcher.bStop {
				return
			}
			msg := <-ch
			for _, ev := range msg.Events {
				if ev.Type == clientv3.EventTypePut {
					nid := string(ev.Kv.Key)
					mpNode := Decode(ev.Kv.Value)
					if mpNode["Nid"] != "" && mpNode["Nid"] == nid {
						node := Node{
							Ndc:  mpNode["Ndc"],
							Nid:  mpNode["Nid"],
							Name: mpNode["Name"],
							Npay: mpNode["Npay"],
						}

						serviceWatcher.nodeLook.Lock()
						serviceWatcher.nodes[nid] = node
						serviceWatcher.nodeLook.Unlock()

						log.Printf("Node Up [%s]", node.Nid)
						if serviceWatcher.callback != nil {
							serviceWatcher.callback(ServerUp, node)
						}
					}
				}
				if ev.Type == clientv3.EventTypeDelete {
					nid := string(ev.Kv.Key)
					node, find := serviceWatcher.GetNodeByID(nid)
					if find {
						log.Printf("Node Down [%s]", node.Nid)
						if serviceWatcher.callback != nil {
							serviceWatcher.callback(ServerDown, *node)
						}
						serviceWatcher.DeleteNodesByID(nid)
					}
				}
			}
		}
	}()
}

// WatchServiceNode 监控指定服务名称的所有服务节点的状态
func (serviceWatcher *ServiceWatcher) WatchServiceNode(prefix string, callback ServiceWatchCallback) {
	serviceWatcher.callback = callback
	serviceWatcher.GetServiceNodes(prefix)
	serviceWatcher.etcd.Watch(prefix, serviceWatcher.WatchNode, true)
}

// GetServiceNodes 获取已经存在节点
func (serviceWatcher *ServiceWatcher) GetServiceNodes(prefix string) {
	rsp, err := serviceWatcher.etcd.GetResponseByPrefix(prefix)
	if err != nil {
		log.Println(err.Error())
	}

	for _, val := range rsp.Kvs {
		mpNode := Decode(val.Value)
		if mpNode["Nid"] != "" {
			node := Node{
				Ndc:  mpNode["Ndc"],
				Nid:  mpNode["Nid"],
				Name: mpNode["Name"],
				Npay: mpNode["Npay"],
			}

			serviceWatcher.nodeLook.Lock()
			serviceWatcher.nodes[node.Nid] = node
			serviceWatcher.nodeLook.Unlock()

			log.Printf("Find Node [%s]", node.Nid)
			if serviceWatcher.callback != nil {
				serviceWatcher.callback(ServerUp, node)
			}
		}
	}
}
