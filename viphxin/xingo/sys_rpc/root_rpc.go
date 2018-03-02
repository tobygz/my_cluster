package sys_rpc

import (
	"github.com/viphxin/xingo/clusterserver"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
)

type RootRpc struct {
}

func (this *RootRpc) GetRpcMap() map[string]func(iface.IRpcRequest) {
	return map[string]func(iface.IRpcRequest){
		"TakeProxy": this.TakeProxy,
	}
}

/*
子节点连上来的通知
*/
func (this *RootRpc) TakeProxy(request iface.IRpcRequest) {
	name := request.GetParam()
	logger.Info("child node " + name + " connected to " + utils.GlobalObject.Name)
	//加到childs并且绑定链接connetion对象
	clusterserver.GlobalClusterServer.AddChild(name, request.GetWriter())
}
