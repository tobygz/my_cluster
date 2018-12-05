package cluster

import (
	"errors"
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"math/rand"
	"strings"
	"sync"
)

type Child struct {
	name string
	rpc  *XingoRpc

	callSwitch bool
}

func NewChild(name string, conn iface.IWriter) *Child {
	return &Child{
		name: name,
		rpc:  NewXingoRpc(conn),

		callSwitch: true,
	}
}

func (this *Child) GetName() string {
	return this.name
}

func (this *Child) CallChildNotForResult(target string, param string, pid uint64, msgid uint32, binData []byte) error {
	return this.rpc.CallRpcNotForResult(target, param, pid, msgid, binData)
}

func (this *Child) CallChildForResult(target string, param string, pid uint32, msgid uint32, binData []byte) (*RpcData, error) {
	return this.rpc.CallRpcForResult(target, param, pid, msgid, binData)
}

func (this *Child) CallChildSucc(target string, param string, pid uint32, msgid uint32, binData []byte) (*RpcData, error) {
	return this.rpc.CallRpcSucc(target, param, pid, msgid, binData)
}

type ChildMgr struct {
	childs map[string]*Child
	sync.RWMutex
}

func NewChildMgr() *ChildMgr {
	return &ChildMgr{
		childs: make(map[string]*Child, 0),
	}
}

func (this *ChildMgr) AddChild(name string, conn iface.IWriter) {
	this.Lock()
	defer this.Unlock()
	if name == "" {
		panic("addchild failed")
	}

	this.childs[name] = NewChild(name, conn)
	logger.Debug(fmt.Sprintf("child %s connected.", name))
}

func (this *ChildMgr) RemoveChild(name string) {
	this.Lock()
	delete(this.childs, name)
	this.Unlock()
	logger.Debug(fmt.Sprintf("child %s lostconnection.", name))
}

func (this *ChildMgr) GetChild(name string) (*Child, error) {
	this.RLock()
	defer this.RUnlock()

	child, ok := this.childs[name]
	if ok {
		return child, nil
	} else {
		return nil, errors.New(fmt.Sprintf("no child named %s", name))
	}
}

func (this *ChildMgr) OperCallSwitch(name string, status bool) {
	this.Lock()
	if c, ok := this.childs[name]; ok {
		c.callSwitch = status
		logger.Infof("child %s call_switch status %b", name, status)
	}
	this.Unlock()
	return
}

func (this *ChildMgr) GetChildsByPrefix(namePrefix string) []*Child {
	this.RLock()
	childs := make([]*Child, 0)
	for k, v := range this.childs {
		if strings.HasPrefix(k, namePrefix) && v.callSwitch {
			childs = append(childs, v)
		}
	}
	this.RUnlock()
	return childs
}

func (this *ChildMgr) GetChilds() []*Child {
	this.RLock()
	childs := make([]*Child, 0)
	for _, v := range this.childs {
		if v.callSwitch {
			childs = append(childs, v)
		}
	}
	this.RUnlock()
	return childs
}

func (this *ChildMgr) GetRandomChild(namesuffix string) *Child {
	childs := make([]*Child, 0)
	if namesuffix != "" {
		//一类
		childs = this.GetChildsByPrefix(namesuffix)
	} else {
		//所有
		childs = this.GetChilds()
	}
	if len(childs) > 0 {
		pos := rand.Intn(len(childs))
		return childs[pos]
	}
	return nil
}

func (this *ChildMgr) String() string {
	var s string
	this.RLock()
	for n, c := range this.childs {
		s = s + fmt.Sprintf("%s: %t;", n, c.callSwitch)
	}
	this.RUnlock()
	return s
}
