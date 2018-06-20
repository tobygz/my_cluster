package sys_rpc

import (
	"fmt"
	"github.com/viphxin/xingo/cluster"
	"github.com/viphxin/xingo/clusterserver"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"os"
	"strings"
	"sync"
)

type MasterRpc struct {
}

func (this *MasterRpc) GetRpcMap() map[string]func(iface.IRpcRequest) {
	return map[string]func(iface.IRpcRequest){
		"TakeProxy": this.TakeProxy,
		"Shutdown":  this.Shutdown,
	}
}

func (this *MasterRpc) TakeProxy(request iface.IRpcRequest) {
	name := request.GetParam()
	logger.Info("node " + name + " connected to master.")
	//加到childs并且绑定链接connetion对象
	clusterserver.GlobalMaster.AddNode(name, request.GetWriter())

	//返回需要链接的父节点
	remotes, err := clusterserver.GlobalMaster.Cconf.GetRemotesByName(name)
	if err == nil {
		var response string
		for _, r := range remotes {
			if _, ok := clusterserver.GlobalMaster.OnlineNodes[r]; ok {
				//父节点在线
				response = fmt.Sprintf("%s,%s", response, r)
			}
		}
		request.SetResult(response)
	}
	//通知当前节点的子节点链接当前节点
	for _, child := range clusterserver.GlobalMaster.Childs.GetChilds() {
		//遍历所有子节点,观察child节点的父节点是否包含当前节点
		remotes, err := clusterserver.GlobalMaster.Cconf.GetRemotesByName(child.GetName())
		if err == nil {
			for _, rname := range remotes {
				if rname == name {
					//包含，需要通知child节点连接当前节点
					//rpc notice
					child.CallChildNotForResult("RootTakeProxy", name, uint64(0), uint32(0), nil)
					break
				}
			}
		}
	}
	return
}

func (this *MasterRpc) Shutdown(request iface.IRpcRequest) {
	name := request.GetParam()
	logger.Info("node " + name + " says shutdown.")

	utils.GlobalObject.IsClose = true

	var wait sync.WaitGroup
	for _, child := range clusterserver.GlobalMaster.Childs.GetChilds() {
		if strings.Contains(child.GetName(), "gate") {
			wait.Add(1)
			go func(child *cluster.Child) {
				logger.Info("shutdown node :" + child.GetName())
				child.CallChildSucc("Doshutdown", "master", uint32(0), uint32(0), nil)
				wait.Done()
			}(child)
		}
	}
	//time.AfterFunc(time.Second*5, func() {
	for _, child := range clusterserver.GlobalMaster.Childs.GetChilds() {
		if strings.Contains(child.GetName(), "gate") == false {
			wait.Add(1)
			go func(child *cluster.Child) {
				logger.Info("shutdown node :" + child.GetName())
				child.CallChildSucc("Doshutdown", "master", uint32(0), uint32(0), nil)
				wait.Done()
			}(child)
		}
	}

	wait.Wait()
	clusterserver.GlobalClusterServer.OnClose()
	logger.Flush()
	os.Exit(0)
	//})
}
