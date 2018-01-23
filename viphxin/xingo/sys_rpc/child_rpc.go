package sys_rpc

import (
	"fmt"
	"github.com/viphxin/xingo/cluster"
	"github.com/viphxin/xingo/clusterserver"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
)

type ChildRpc struct {
}

/*
master 通知父节点上线, 收到通知的子节点需要链接对应父节点
*/
func (this *ChildRpc) RootTakeProxy(request *cluster.RpcRequest) {
	rname := request.Rpcdata.Param
	logger.Info(fmt.Sprintf("root node %s online. connecting...", rname))
	clusterserver.GlobalClusterServer.ConnectToRemote(rname)
}

func (this *ChildRpc) Doshutdown(request *cluster.RpcRequest) {
	rname := request.Rpcdata.Param
	//utils.GlobalObject.OnServerStop()
	utils.GlobalObject.IsClose = true
	logger.Info(fmt.Sprintf("root node %s send shutdown...", rname))
	clusterserver.GlobalClusterServer.OnClose()
}
