package sys_rpc

import (
	"fmt"
	"github.com/viphxin/xingo/clusterserver"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
)

type ChildRpc struct {
}

func (this *ChildRpc) GetRpcMap() map[string]func(iface.IRpcRequest) {
	return map[string]func(iface.IRpcRequest){
		"RootTakeProxy": this.RootTakeProxy,
		"Doshutdown":    this.Doshutdown,
	}
}

/*
master 通知父节点上线, 收到通知的子节点需要链接对应父节点
*/
func (this *ChildRpc) RootTakeProxy(request iface.IRpcRequest) {
	rname := request.GetParam()
	logger.Info("root node", rname, "online. connecting rpcdata:", request.GetData())
	clusterserver.GlobalClusterServer.ConnectToRemote(rname)
}

func (this *ChildRpc) Doshutdown(request iface.IRpcRequest) {
	rname := request.GetParam()
	//utils.GlobalObject.OnServerStop()
	utils.GlobalObject.IsClose = true
	logger.Info(fmt.Sprintf("root node %s send shutdown...", rname))
	clusterserver.GlobalClusterServer.OnClose()
}
