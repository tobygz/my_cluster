package cluster

import (
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"io"
)

type RpcServerProtocol struct {
	rpcMsgHandle *RpcMsgHandle
	rpcDatapack  *RpcDataPack
}

func NewRpcServerProtocol() *RpcServerProtocol {
	return &RpcServerProtocol{
		rpcMsgHandle: NewRpcMsgHandle(),
		rpcDatapack:  NewRpcDataPack(),
	}
}

func (this *RpcServerProtocol) GetMsgHandle() iface.Imsghandle {
	return this.rpcMsgHandle
}

func (this *RpcServerProtocol) GetDataPack() iface.Idatapack {
	return this.rpcDatapack
}

func (this *RpcServerProtocol) ManualMsgPush(msgId uint32, data []byte, pid uint64, fconn iface.Iconnection) {
	logger.Info("invalid ManualMsgPush rpcprotocol, fatal error !")
}

func (this *RpcServerProtocol) AddRpcRouter(router interface{}) {
	this.rpcMsgHandle.AddRpcRouter(router.(iface.IRpcRouter))
}

func (this *RpcServerProtocol) InitWorker(poolsize int32) {
	logger.Debug("called StartWorkerLoop InitWorker 111")
	this.rpcMsgHandle.StartWorkerLoop(int(poolsize))
}

func (this *RpcServerProtocol) OnConnectionMade(fconn iface.Iconnection) {
	logger.Info(fmt.Sprintf("node ID: %d connected. IP Address: %s", fconn.GetSessionId(), fconn.RemoteAddr()))
	utils.GlobalObject.OnClusterConnectioned(fconn)
}

func (this *RpcServerProtocol) OnConnectionLost(fconn iface.Iconnection) {
	logger.Info(fmt.Sprintf("node ID: %d disconnected. IP Address: %s", fconn.GetSessionId(), fconn.RemoteAddr()))
	utils.GlobalObject.OnClusterClosed(fconn)
}

func (this *RpcServerProtocol) StartReadThread(fconn iface.Iconnection) {
	logger.Debug("start receive rpc data from socket...")
	for {
		//read per head data
		headdata := make([]byte, this.rpcDatapack.GetHeadLen())

		if _, err := io.ReadFull(fconn.GetConnection(), headdata); err != nil {
			logger.Error(err)
			fconn.Stop()
			return
		}
		pkgHead, err := this.rpcDatapack.Unpack(headdata, nil)
		if err != nil {
			logger.Error(err)
			fconn.Stop()
			return
		}
		//data
		pkg := pkgHead.(*RpcPackege)
		if pkg.Len > 0 {
			pkg.Data = make([]byte, pkg.Len)
			if _, err := io.ReadFull(fconn.GetConnection(), pkg.Data); err != nil {
				logger.Error(err)
				fconn.Stop()
				return
			} else {
				rpcRequest := &RpcRequest{
					Fconn:   fconn,
					Rpcdata: &RpcData{},
				}

				if !rpcRequest.Rpcdata.Decode(pkg.Data) {
					logger.Infof("parse rpcdata failed.")
					fconn.Stop()
					return
				}

				//logger.Trace(fmt.Sprintf("rpc call server. data len: %d. MsgType: %d", pkg.Len, int(rpcRequest.Rpcdata.MsgType)))
				if utils.GlobalObject.PoolSize > 0 && rpcRequest.Rpcdata.MsgType != RESPONSE {
					this.rpcMsgHandle.DeliverToMsgQueue(rpcRequest)
				} else {
					this.rpcMsgHandle.DoMsgFromGoRoutine(rpcRequest)
				}
			}
		}
	}
}

type RpcClientProtocol struct {
	rpcMsgHandle *RpcMsgHandle
	rpcDatapack  *RpcDataPack
}

func NewRpcClientProtocol() *RpcClientProtocol {
	return &RpcClientProtocol{
		rpcMsgHandle: NewRpcMsgHandle(),
		rpcDatapack:  NewRpcDataPack(),
	}
}

func (this *RpcClientProtocol) GetMsgHandle() iface.Imsghandle {
	return this.rpcMsgHandle
}

func (this *RpcClientProtocol) GetDataPack() iface.Idatapack {
	return this.rpcDatapack
}

func (this *RpcClientProtocol) AddRpcRouter(router interface{}) {
	this.rpcMsgHandle.AddRpcRouter(router.(iface.IRpcRouter))
}

func (this *RpcClientProtocol) InitWorker(poolsize int32) {
	logger.Debug("called StartWorkerLoop InitWorker 000")
	this.rpcMsgHandle.StartWorkerLoop(int(poolsize))
}

func (this *RpcClientProtocol) OnConnectionMade(fconn iface.Iclient) {
	utils.GlobalObject.OnClusterCConnectioned(fconn)
}

func (this *RpcClientProtocol) OnConnectionLost(fconn iface.Iclient) {
	utils.GlobalObject.OnClusterCClosed(fconn)
}

func (this *RpcClientProtocol) StartReadThread(fconn iface.Iclient) {
	logger.Debug("start receive rpc data from socket...")
	for {
		//read per head data
		headdata := make([]byte, this.rpcDatapack.GetHeadLen())

		if _, err := io.ReadFull(fconn.GetConnection(), headdata); err != nil {
			logger.Error(err)
			fconn.Stop(false)
			return
		}
		pkgHead, err := this.rpcDatapack.Unpack(headdata, nil)
		if err != nil {
			fconn.Stop(false)
			return
		}
		//data
		pkg := pkgHead.(*RpcPackege)
		if pkg.Len > 0 {
			pkg.Data = make([]byte, pkg.Len)
			if _, err := io.ReadFull(fconn.GetConnection(), pkg.Data); err != nil {
				fconn.Stop(false)
				return
			} else {
				rpcRequest := &RpcRequest{
					Fconn:   fconn,
					Rpcdata: &RpcData{},
				}

				rpcRequest.Rpcdata.Decode(pkg.Data)

				//logger.Trace(fmt.Sprintf("rpc call client. data len: %d. MsgType: %d", pkg.Len, rpcRequest.Rpcdata.MsgType))
				if utils.GlobalObject.PoolSize > 0 && rpcRequest.Rpcdata.MsgType != RESPONSE {
					this.rpcMsgHandle.DeliverToMsgQueue(rpcRequest)
				} else {
					this.rpcMsgHandle.DoMsgFromGoRoutine(rpcRequest)
				}
			}
		}
	}
}
