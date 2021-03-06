package fnet

import (
	"bytes"
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
	lastTick     uint32
	gateChild    iface.IChild
}

func NewConnection(conn *net.TCPConn, sessionId uint32, protoc iface.IServerProtocol) *Connection {
	fconn := &Connection{
		Conn:        conn,
		isClosed:    false,
		SessionId:   sessionId,
		Protoc:      protoc,
		lastTick:    utils.GetFastSec(),
		PropertyBag: make(map[string]interface{}),
		//SendBuffChan: make(chan []byte, utils.GlobalObject.MaxSendChanLen),
		SendBuffChan: make(chan []byte, 10240),
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
	if utils.GlobalObject.IsMaster() == false {
		//this.StartWriteThread()
	} else {
		this.StartWriteThread1()
	}
	this.Protoc.StartReadThread(this)
}

func (this *Connection) GetLastTick() uint32 {
	return this.lastTick
}

func (this *Connection) UpdateLastTick() {
	this.lastTick = utils.GetFastSec()
}

func (this *Connection) Stop() {
	// 防止将Send放在go内造成的多线程冲突问题
	this.sendtagGuard.Lock()
	defer this.sendtagGuard.Unlock()

	logger.Error(fmt.Sprintf("raw fnet connection stop sessid: %d", this.SessionId))
	if this.isClosed {
		return
	}
	this.isClosed = true

	this.ExtSendChan <- true
	this.isClosed = true
	//掉线回调放到go内防止，掉线回调处理出线死锁
	go this.Protoc.OnConnectionLost(this)
	//remove to connectionmsg
	//utils.GlobalObject.TcpServer.GetConnectionMgr().Remove(this)
	close(this.ExtSendChan)
	close(this.SendBuffChan)

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

func (this *Connection) Send(data []byte) error {
	// 防止将Send放在go内造成的多线程冲突问题
	//this.sendtagGuard.Lock()
	//defer this.sendtagGuard.Unlock()

	if !this.isClosed {
		if _, err := this.Conn.Write(data); err != nil {
			logger.Error(fmt.Sprintf("send data error.reason: %s", err))
			return err
		}
		return nil
	} else {
		return errors.New("connection closed")
	}
}

func (this *Connection) SendBuff(data []byte) error {
	// 防止将Send放在go内造成的多线程冲突问题
	this.sendtagGuard.Lock()
	defer this.sendtagGuard.Unlock()

	if !this.isClosed {

		// 发送超时
		select {
		case <-time.After(time.Second * 2):
			logger.Error("send error: timeout.")
			return errors.New("send error: timeout.")
		case this.SendBuffChan <- data:
			return nil
		}
	} else {
		return errors.New("connection closed")
	}

}

func (this *Connection) RemoteAddr() net.Addr {
	return (*this.Conn).RemoteAddr()
}

func (this *Connection) LostConnection() {
	(*this.Conn).Close()
	this.isClosed = true
	logger.Info("LostConnection==============!!!!!!!!!!!!!!!!!!!!!!!!!")
}

func getMs() int64 {
	return time.Now().UnixNano() / 1000000
}
func (this *Connection) StartWriteThread() {
	go func() {
		logger.Debug("start send data from channel...")
		total := uint32(0)
		nowMs := getMs()
		lMs := getMs()

		nowSec := utils.GetFastSec()
		lSec := nowSec
		pkg_chan := make(chan []byte, 102400)
		for {
			data := <-this.SendBuffChan

			pkg_chan <- data
			nowMs = getMs()
			if nowMs-lMs < 50 && false {
				continue
			}
			lMs = nowMs
			logger.Fatal("CallRpcNotForResult conn consume sendbuff len: ", len(this.SendBuffChan), " now send:", len(pkg_chan))

			slc := make([][]byte, len(pkg_chan))
			for index := 0; index < len(pkg_chan); index++ {
				slc[index] = <-pkg_chan
			}
			sep := []byte("")
			mem := bytes.Join(slc, sep)

			n, err := this.Conn.Write(mem)
			if err != nil {
				logger.Info("send data error exit...")
				return
			}
			total = total + uint32(n)

			nowSec = utils.GetFastSec()
			if nowSec-lSec >= 1 {
				logger.Fatal("connection sendio len: ", total, " sendbuflen:", len(this.SendBuffChan))
				lSec = nowSec
				total = 0
			}
		}
	}()
}

func (this *Connection) StartWriteThread1() {
	go func() {
		logger.Debug("start send data from channel...")
		for {
			select {
			case <-this.ExtSendChan:
				logger.Info("send thread exit successful!!!!")
				return
			case data := <-this.SendBuffChan:
				//send
				if _, err := this.Conn.Write(data); err != nil {
					logger.Info("send data error exit...")
					return
				}
			}
		}
	}()
}
