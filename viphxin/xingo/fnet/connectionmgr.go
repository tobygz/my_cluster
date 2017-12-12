package fnet

import (
	"errors"
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"sync"
)

type ConnectionMgr struct {
	connections map[uint32]iface.Iconnection
	conMrgLock  sync.RWMutex
}

func (this *ConnectionMgr) Add(conn iface.Iconnection) {
	this.conMrgLock.Lock()
	defer this.conMrgLock.Unlock()
	this.connections[conn.GetSessionId()] = conn
	logger.Debug(fmt.Sprintf("Total connection: %d", len(this.connections)))
}

func (this *ConnectionMgr) Remove(conn iface.Iconnection) error {
	this.conMrgLock.Lock()
	defer this.conMrgLock.Unlock()
	ssid := conn.GetSessionId()
	_, ok := this.connections[ssid]
	if ok {
		delete(this.connections, ssid)
		logger.Info(len(this.connections), "del ssid: ", ssid)
		return nil
	} else {
		return errors.New("not found!!")
	}
}

func (this *ConnectionMgr) Get(sid uint32) (iface.Iconnection, error) {
	this.conMrgLock.Lock()
	defer this.conMrgLock.Unlock()
	v, ok := this.connections[sid]
	if ok {
		//delete(this.connections, sid)
		return v, nil
	} else {
		return nil, errors.New("not found!!")
	}
}

func (this *ConnectionMgr) Len() int {
	this.conMrgLock.Lock()
	defer this.conMrgLock.Unlock()
	return len(this.connections)
}

func NewConnectionMgr() *ConnectionMgr {
	return &ConnectionMgr{
		connections: make(map[uint32]iface.Iconnection),
	}
}
