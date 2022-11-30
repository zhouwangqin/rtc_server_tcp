package etcd

import (
	"encoding/json"
)

// Node 服务节点对象
type Node struct {
	// Ndc 节点区域
	Ndc string
	// Nid 节点id
	Nid string
	// Name 节点名
	Name string
	// Npay 节点负载
	Npay string
}

// GetNodeValue 获取节点保存的值
func (node *Node) GetNodeValue() string {
	data := make(map[string]string)
	data["Ndc"] = node.Ndc
	data["Nid"] = node.Nid
	data["Name"] = node.Name
	data["Npay"] = node.Npay
	return Encode(data)
}

// Encode 将map格式转换成string
func Encode(data map[string]string) string {
	if data != nil {
		str, _ := json.Marshal(data)
		return string(str)
	}
	return ""
}

// Decode 将string格式转换成map
func Decode(str []byte) map[string]string {
	if len(str) > 0 {
		var data map[string]string
		json.Unmarshal(str, &data)
		return data
	}
	return nil
}

// GetRPCChannel 获取RPC对象string
func GetRPCChannel(node Node) string {
	return "rpc-" + node.Nid
}

// GetEventChannel 获取广播对象string
func GetEventChannel(node Node) string {
	return "event-" + node.Nid
}
