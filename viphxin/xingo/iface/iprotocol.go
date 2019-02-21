package iface

type IClientProtocol interface {
	OnConnectionMade(fconn Iclient)
	OnConnectionLost(fconn Iclient)
	StartReadThread(fconn Iclient)
	InitWorker(int32)
	AddRpcRouter(interface{})
	GetMsgHandle() Imsghandle
	GetDataPack() Idatapack
}

type IServerProtocol interface {
	OnConnectionMade(fconn Iconnection)
	OnConnectionLost(fconn Iconnection)
	StartReadThread(fconn Iconnection)
	InitRc4([]byte)
	InitWorker(int32)
	AddRpcRouter(interface{})
	GetMsgHandle() Imsghandle
	GetDataPack() Idatapack
	ManualMsgPush(msgId uint32, data []byte, pid uint64, fconn Iconnection)
}
