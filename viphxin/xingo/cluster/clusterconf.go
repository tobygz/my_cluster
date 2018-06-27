package cluster

import (
	"encoding/json"
	"errors"
	"io/ioutil"
)

type ClusterServerConf struct {
	Name      string
	Host      string
	RootPort  int
	KcpIs     bool
	UdpPort   int
	UdpIp     string
	Http      []interface{} //[port, staticfile_path]
	Https     []interface{} //[port, certFile, keyFile, staticfile_path]
	NetPort   int
	Remotes   []string
	Module    string
	Db        string
	Dbid      string
	Dbpwd     string
	Startid   uint64
	Ridamount uint32
	PProfAddr string
	Log       string
}

type ClusterConf struct {
	Master  *ClusterServerConf
	Servers map[string]*ClusterServerConf
}

func NewClusterConf(path string) (*ClusterConf, error) {
	cconf := &ClusterConf{}
	//集群服务器配置信息
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, cconf)
	if err != nil {
		panic(err)
	}

	return cconf, nil
}

func (this *ClusterConf) GetPProfAddr(name string) string {
	conf, ok := this.Servers[name]
	if ok {
		return conf.PProfAddr
	}
	return ""
}

/*
获取当前节点的父节点
*/
func (this *ClusterConf) GetRemotesByName(name string) ([]string, error) {
	server, ok := this.Servers[name]
	if ok {
		return server.Remotes, nil
	} else {
		return nil, errors.New("no server found!!!")
	}
}

/*
获取当前节点的子节点
*/
func (this *ClusterConf) GetChildsByName(name string) []string {
	names := make([]string, 0)
	for sername, ser := range this.Servers {
		for _, rname := range ser.Remotes {
			if rname == name {
				names = append(names, sername)
				break
			}
		}
	}
	return names
}
