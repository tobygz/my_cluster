package iface

type IDbop interface {
	Start(dbid string, addr string, pwd string)
	DoDbTask(opCmd string, args ...interface{}) (interface{}, error)
}
