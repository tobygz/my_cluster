package fnet

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
)

type ConnectionMgr struct {
	connections map[uint32]iface.Iconnection
	conMrgLock  sync.RWMutex
	AllKeys     chan []byte
}

func (this *ConnectionMgr) FetchEncKey() []byte {
	val := <-this.AllKeys
	if len(this.AllKeys) <= 0 {
		this.Init()
	}
	return val
}

func (this *ConnectionMgr) Init() {
	logger.Info("ConnectionMgr begin init")
	this.AllKeys = geneRandKeyChan(32, 10240)
	logger.Info("ConnectionMgr after init")
}

func geneRandKeyChan(l int, lenv int) chan []byte {
	rand.Seed(time.Now().UnixNano())
	slcKey := make(chan []byte, lenv)
	for j := 0; j < lenv; j++ {
		slc := make([]byte, 0, l)
		for i := 0; i < l; i++ {
			x := rand.Intn(255)
			slc = append(slc, byte(x))
		}
		slcKey <- slc
	}
	return slcKey
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

func NewConnectionMgr(bnet bool) *ConnectionMgr {
	obj := &ConnectionMgr{
		connections: make(map[uint32]iface.Iconnection),
	}
	if bnet {
		obj.Init()
	}
	return obj
}
