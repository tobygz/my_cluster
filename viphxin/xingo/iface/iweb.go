package iface

import ()

type Iweb interface {
	StartParseReq()
	AddHandles(router interface{})
	Start(port string)
	RawClose()
	HandleReqCall(req interface{})
	GetReqChan() chan interface{}
}
