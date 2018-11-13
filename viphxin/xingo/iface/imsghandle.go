package iface

type Imsghandle interface {
	DeliverToMsgQueue(interface{})
	DoMsgFromGoRoutine(interface{})
	AddRouter(IRouter)
	AddRpcRouter(IRpcRouter)
	StartWorkerLoop(int)
	UpdateNetIn(int)
	UpdateNetOut(int)
	Name() string
}
