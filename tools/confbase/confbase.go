package confbase

import (
	"github.com/whatap/kube/tools/lang/value"
)

func GetConfig(item string, host string, port int, callback func(key string, val string)) error {
	p := value.NewMapValue()
	p.PutString("cmd", "get")
	p.PutString("id", item)
	p.PutLong("filetime", int64(0))

	ret, err := sendmap(host, port, p)
	if err != nil {
		return err
	}
	if ret.GetBool("ok") {
		conf := ret.GetMap("config")
		if conf != nil {
			conf.IterateString(func(key string, val string) {
				callback(key, val)
			})
		}
	}
	return nil
}

func GetNodeAccess(host string, port int, nodeName string, h2 func(string, int32)) (err error) {
	p := value.NewMapValue()
	p.PutString("cmd", "regist")
	p.Put("kube.micro", value.NewBoolValue(true))
	p.Put("onode_name", value.NewTextValue(nodeName))
	p.Put("oid", value.NewIntValue(int32(9999)))

	ret, err := sendmap(host, port, p)
	if err != nil {
		return err
	}
	if ret.GetBool("ok") {
		nodeAgentIp := ret.GetString("node.agent.ip")
		nodeAgentPort := ret.GetInt("node.agent.port")
		h2(nodeAgentIp, int32(nodeAgentPort))
	}

	return nil

}

func GetContainerPerf(nodeIP string, nodePort int, containerId string, h2 func(key string, val string)) (err error) {
	p := value.NewMapValue()
	p.PutString("container_id", containerId)

	ret, err := sendmap(nodeIP, nodePort, p)
	if err != nil {
		return err
	}
	keys := ret.Keys()
	for keys.HasMoreElements() {
		key := keys.NextString()
		val := ret.Get(key)

		h2(key, val.ToString())
	}

	return
}
