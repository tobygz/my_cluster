package iface

import (
	"net"
)

type IChild interface {
	CallChildNotForResult(target string, args ...interface{}) error
}

type Iconnection interface {
	Start()
	Stop()
	GetConnection() *net.TCPConn
	GetSessionId() uint32
	Send([]byte) error
	SendBuff([]byte) error
	GetLastTick() uint32
	UpdateLastTick()
	RemoteAddr() net.Addr
	LostConnection()
	GetProperty(string) (interface{}, error)
	SetProperty(string, interface{})
	RemoveProperty(string)
}
