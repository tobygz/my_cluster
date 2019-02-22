package fnet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/viphxin/xingo/encry"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
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

	//encry
	RsaObj *encry.RsaCipher
	rc4Key []byte
	rc4Alg *encry.Cipher
	bEnc   bool
}

var pub_key string = "pem/client_public.pem"
var pri_key string = "pem/server_private.pem"

func NewConnection(conn *net.TCPConn, sessionId uint32, protoc iface.IServerProtocol, key []byte) *Connection {
	fconn := &Connection{
		Conn:         conn,
		isClosed:     false,
		SessionId:    sessionId,
		Protoc:       protoc,
		bufobj:       bufio.NewWriterSize(conn, 1024),
		PropertyBag:  make(map[string]interface{}),
		SendBuffChan: make(chan []byte, 1), //utils.GlobalObject.MaxSendChanLen),
		ExtSendChan:  make(chan bool, 1),
		rc4Key:       key,
		bEnc:         false,
	}
	fconn.rc4Alg, _ = encry.NewCipher(key)
	if key != nil && len(key) != 0 {
		fconn.bEnc = true
		fconn.RsaObj = encry.NewRsaCipher(pub_key, pri_key)
	}
	//set  connection time
	fconn.SetProperty("xingo_ctime", time.Since(time.Now()))
	return fconn
}

func (this *Connection) GetRc4() (interface{}, bool) {
	return this.rc4Alg, this.bEnc
}

func (this *Connection) SyncKey() bool {
	logger.Infof("Connection SyncKey rc4key:", this.rc4Key)
	if this.rc4Key == nil || len(this.rc4Key) == 0 {
		return true
	}
	encRsaData := this.RsaObj.Encrypt(this.rc4Key)
	logger.Infof("enc len: %d data: %v", len(encRsaData), encRsaData)
	ob := make([]byte, uint32(len(encRsaData)+4))
	binary.LittleEndian.PutUint32(ob[0:], uint32(len(encRsaData)))
	copy(ob[4:], encRsaData)
	this.Conn.Write(ob)
	logger.Infof("write len: %d data: %v", len(encRsaData), encRsaData)

	lenvSlc := make([]byte, 4)
	if _, err := io.ReadFull(this.Conn, lenvSlc); err != nil {
		logger.Errorf("SyncKey rc4key err:%v", err)
		return false
	}
	reader := bytes.NewReader(lenvSlc)

	lenv := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &lenv); err != nil {
		logger.Errorf("SyncKey rc4key nil err:", err)
		return false
	}

	cont := make([]byte, lenv)
	if _, err := io.ReadFull(this.Conn, cont); err != nil {
		logger.Errorf("SyncKey rc4key nil err:", err)
		return false
	}
	decData, err := this.RsaObj.Decrypt(cont)
	if err != nil {
		logger.Errorf("SyncKey rsa decrypt failed, cont: %v", cont)
		return false
	}

	if len(decData) != len(this.rc4Key) {
		logger.Errorf("SyncKey rc4key len not match, len decdata: %d rc4key len: %d", len(decData), len(this.rc4Key))
		return false
	}

	for i := 0; i < len(decData); i++ {
		if decData[i] != this.rc4Key[i] {
			logger.Errorf("SyncKey rc4key: %v cont: %v", this.rc4Key, decData)
			return false
		}
	}
	logger.Infof("synckey check succ genid: %d", this.SessionId)
	return true
}

func (this *Connection) Start() {
	logger.Infof("connection started")
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
	this.sendtagGuard.Lock()
	if this.isClosed {
		this.sendtagGuard.Unlock()
		return
	} else {
		this.isClosed = true
	}
	this.sendtagGuard.Unlock()

	logger.Error(fmt.Sprintf("raw fnet connection stop sessid: %d", this.SessionId))

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

func (this *Connection) RawSend(data []byte) error {
	if !this.isClosed {
		this.SendBuffChan <- data
		return nil
	} else {
		return errors.New("connection closed")
	}

}
func (this *Connection) Send(data []byte) error {
	this.sendtagGuard.Lock()
	defer this.sendtagGuard.Unlock()

	if this.bEnc {
		encData := make([]byte, len(data)-8)
		this.rc4Alg.XorKeyStreamGeneric(encData, data[8:])
		this.RawSend(data[:8])
		this.RawSend(encData)
		logger.Infof("raw send data: %v encData: %v", data, encData)
	} else {
		this.RawSend(data)
	}
	return nil
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
