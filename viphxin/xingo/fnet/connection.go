package fnet

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"net"
	"sync"
	"time"
)

type Connection struct {
	Conn         *net.TCPConn
	isClosed     bool
	SessionId    uint32
	Protoc       iface.IServerProtocol
	PropertyBag  map[string]interface{}
	sendtagGuard sync.RWMutex
	propertyLock sync.RWMutex

	SendBuffChan chan []byte
	ExtSendChan  chan bool
	bufobj       *bufio.Writer
}

func NewConnection(conn *net.TCPConn, sessionId uint32, protoc iface.IServerProtocol) *Connection {
	fconn := &Connection{
		Conn:         conn,
		isClosed:     false,
		SessionId:    sessionId,
		Protoc:       protoc,
		bufobj:       bufio.NewWriterSize(conn, 1024),
		PropertyBag:  make(map[string]interface{}),
		SendBuffChan: make(chan []byte, 1), //utils.GlobalObject.MaxSendChanLen),
		ExtSendChan:  make(chan bool, 1),
	}
	//set  connection time
	fconn.SetProperty("xingo_ctime", time.Since(time.Now()))
	return fconn
}

func (this *Connection) Start() {
	//add to connectionmsg
	utils.GlobalObject.TcpServer.GetConnectionMgr().Add(this)
	this.Protoc.OnConnectionMade(this)
	//this.StartWriteThread()
	go this.sendThreadLoopMode()
	this.Protoc.StartReadThread(this)
}

func (this *Connection) SendBuff([]byte) error {
	return nil
}

func (this *Connection) Stop() {
	// 防止将Send放在go内造成的多线程冲突问题
	if this.IsClosed() {
		return
	}

	logger.Error(fmt.Sprintf("raw fnet connection stop sessid: %d", this.SessionId))
	this.SetClosed()

	this.ExtSendChan <- true
	this.ExtSendChan <- true
	//掉线回调放到go内防止，掉线回调处理出线死锁
	go this.Protoc.OnConnectionLost(this)
	//remove to connectionmsg
	//utils.GlobalObject.TcpServer.GetConnectionMgr().Remove(this)
	close(this.ExtSendChan)
	close(this.SendBuffChan)

	this.bufobj.Flush()
	this.Conn.Close()
}

func (this *Connection) GetConnection() *net.TCPConn {
	return this.Conn
}

func (this *Connection) GetSessionId() uint32 {
	return this.SessionId
}

func (this *Connection) GetProtoc() iface.IServerProtocol {
	return this.Protoc
}

func (this *Connection) GetProperty(key string) (interface{}, error) {
	this.propertyLock.RLock()
	defer this.propertyLock.RUnlock()

	value, ok := this.PropertyBag[key]
	if ok {
		return value, nil
	} else {
		return nil, errors.New("no property in connection")
	}
}

func (this *Connection) SetProperty(key string, value interface{}) {
	this.propertyLock.Lock()
	defer this.propertyLock.Unlock()

	this.PropertyBag[key] = value
}

func (this *Connection) RemoveProperty(key string) {
	this.propertyLock.Lock()
	defer this.propertyLock.Unlock()

	delete(this.PropertyBag, key)
}

func (this *Connection) sendThread() {
	bflush := false
	for {
		select {
		case <-this.ExtSendChan:
			logger.Info("sendThread exit successful!!!!")
			return
		case data := <-this.SendBuffChan:
			this.bufobj.Write(data)
			utils.GlobalObject.Protoc.GetMsgHandle().UpdateNetOut(len(data))
			bflush = true
		default:
			if bflush == false {
				data := <-this.SendBuffChan
				this.bufobj.Write(data)
				utils.GlobalObject.Protoc.GetMsgHandle().UpdateNetOut(len(data))
				bflush = true
			} else {
				this.bufobj.Flush()
				bflush = false
			}
		}

	}
}

func (this *Connection) sendThreadLoopMode() {
	bNet := utils.GlobalObject.IsNet()
	msgh := utils.GlobalObject.Protoc.GetMsgHandle()
	for {
		select {
		case data := <-this.SendBuffChan:
			this.bufobj.Write(data)
			if bNet {
				if msgh == nil {
					msgh = utils.GlobalObject.Protoc.GetMsgHandle()
				}
				msgh.UpdateNetOut(len(data))
			}
		case <-this.ExtSendChan:
			logger.Info("sendThreadLoopMode exit successful!!!")
			return
		}
	hasData:
		for {
			select {
			case data := <-this.SendBuffChan:
				this.bufobj.Write(data)
				if msgh == nil {
					msgh = utils.GlobalObject.Protoc.GetMsgHandle()
				}
				msgh.UpdateNetOut(len(data))
			case <-this.ExtSendChan:
				logger.Info("sendThreadLoopMode exit successful in loop!!!")
				return
			default:
				this.bufobj.Flush()
				break hasData
			}
		}
	}
}

func (this *Connection) IsClosed() bool {
	this.sendtagGuard.Lock()
	defer this.sendtagGuard.Unlock()
	return this.isClosed
}

func (this *Connection) SetClosed() {
	this.sendtagGuard.Lock()
	defer this.sendtagGuard.Unlock()
	this.isClosed = true
}

func (this *Connection) Send(data []byte) error {
	if !this.IsClosed() {
		this.SendBuffChan <- data
		return nil
	} else {
		return errors.New("connection closed")
	}
}

func (this *Connection) RemoteAddr() net.Addr {
	//func (this *Connection) RemoteAddr() string {
	return (*this.Conn).RemoteAddr()
}

func (this *Connection) LostConnection() {
	(*this.Conn).Close()
	this.isClosed = true
	logger.Info("LostConnection==============!!!!!!!!!!!!!!!!!!!!!!!!!")
}
