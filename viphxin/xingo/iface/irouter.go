package iface

type IRequest interface {
	GetConnection() Iconnection
	GetUdpConn() IUdpConn
	GetData() []byte
	GetMsgId() uint32
}

type IRouter interface {
	GetApiMap() map[string]func(IRequest, uint32, uint32)
	GetGApiMap() map[string]func(IRequest, uint32, uint64)
}

type IRpcRequest interface {
	GetWriter() IWriter
	GetMsgType() uint8
	GetKey() string
	GetTarget() string
	GetParam() string
	GetResult() string
	GetData() []byte
	GetPid() uint64
	GetMsgid() uint32
	SetResult(string)
}

type IRpcRouter interface {
	GetRpcMap() map[string]func(IRpcRequest)
}
