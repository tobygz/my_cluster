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

const (
	MAX_RETRY      = 1024 //父节点掉线最大重连次数
	RETRY_INTERVAL = 60   //重连间隔60s
)

type TcpClient struct {
	conn          *net.TCPConn
	addr          *net.TCPAddr
	protoc        iface.IClientProtocol
	propertyBag   map[string]interface{}
	reconnCB      func(iface.Iclient)
	maxRetry      int
	retryInterval int
	sendtagGuard  sync.RWMutex
	propertyLock  sync.RWMutex
	sendCh        chan []byte
	exitCh        chan bool
	bufobj        *bufio.Writer
	QpsObj        *utils.QpsMgr
}

func NewReConnTcpClient(ip string, port int, protoc iface.IClientProtocol, maxRetry int,
	retryInterval int, reconnCB func(iface.Iclient)) *TcpClient {
	client := NewTcpClient(ip, port, protoc)
	client.maxRetry = maxRetry
	client.retryInterval = retryInterval
	client.reconnCB = reconnCB
	return client
}

func NewTcpClient(ip string, port int, protoc iface.IClientProtocol) *TcpClient {
	addr := &net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
		Zone: "",
	}
	for {
		conn, err := net.DialTCP("tcp", nil, addr)
		if err == nil {
			client := &TcpClient{
				conn:        conn,
				addr:        addr,
				protoc:      protoc,
				propertyBag: make(map[string]interface{}, 0),
				QpsObj:      utils.NewQps(time.Second),
				sendCh:      make(chan []byte, 1),
				exitCh:      make(chan bool, 1),
				bufobj:      bufio.NewWriterSize(conn, 1024),
			}
			go client.protoc.OnConnectionMade(client)
			return client
		} else {
			time.Sleep(time.Second)
		}
	}

}

func (this *TcpClient) Start() {
	go this.sendThread()
	go this.protoc.StartReadThread(this)
}

func (this *TcpClient) Stop(isforce bool) {

	if utils.GlobalObject.IsClose {
		isforce = true
	}

	if this.maxRetry == 0 || isforce {
		this.protoc.OnConnectionLost(this)
	} else {
		//retry
		if this.reconnection() {
			//顺序很重要，先把读数据用的goroutine开启
			this.Start()
			if this.reconnCB != nil {
				this.reconnCB(this)
			}
		}
	}
}

func (this *TcpClient) resetConn(conn *net.TCPConn) {
	this.conn = conn
	this.exitCh = make(chan bool, 1)
	this.bufobj = bufio.NewWriterSize(conn, 1024)
}

func (this *TcpClient) reconnection() bool {
	logger.Info("TcpClient reconnection ...")
	for i := 1; i <= this.maxRetry; i++ {
		logger.Info("retry time ", i)
		conn, err := net.DialTCP("tcp", nil, this.addr)
		if err == nil {
			this.resetConn(conn)
			return true
		} else {
			d, err := time.ParseDuration(fmt.Sprintf("%ds", this.retryInterval))
			if err != nil {
				time.Sleep(RETRY_INTERVAL * time.Second)
			} else {
				time.Sleep(d)
			}
		}
	}
	return false
}

func (this *TcpClient) Send(data []byte) error {
	this.sendCh <- data
	this.QpsObj.Add(1, len(data))
	flag, info, _ := this.QpsObj.Dump()
	if flag {
		logger.Prof(info)
	}
	return nil
}

/*
func (this *TcpClient) sendThread() {
	for {
		data := <-this.sendCh
		n, err := this.conn.Write(data)
		if err != nil {
			logger.Error(fmt.Sprintf("rpc client send data error.reason: %s", err))
			return
		}
		this.QpsObj.Add(1, n)
		flag, info, _ := this.QpsObj.Dump()
		if flag {
			logger.Prof(info)
		}
	}
}
*/

func (this *TcpClient) sendThread() {
	logger.Info("TcpClient sendThread started")
	for {
		select {
		case data := <-this.sendCh:
			this.bufobj.Write(data)
		case <-this.exitCh:
			logger.Info("TcpClient sendThread exit successful!!!")
			return
		}
	hasData:
		for {
			select {
			case data := <-this.sendCh:
				this.bufobj.Write(data)
			case <-this.exitCh:
				logger.Info("TcpClient sendThread exit successful in loop!!!")
				return
			default:
				this.bufobj.Flush()
				break hasData
			}
		}
	}
}

func (this *TcpClient) GetConnection() *net.TCPConn {
	return this.conn
}

func (this *TcpClient) GetProperty(key string) (interface{}, error) {
	this.propertyLock.RLock()
	defer this.propertyLock.RUnlock()

	value, ok := this.propertyBag[key]
	if ok {
		return value, nil
	} else {
		return nil, errors.New("no property in connection")
	}
}

func (this *TcpClient) SetProperty(key string, value interface{}) {
	this.propertyLock.Lock()
	defer this.propertyLock.Unlock()

	this.propertyBag[key] = value
}

func (this *TcpClient) RemoveProperty(key string) {
	this.propertyLock.Lock()
	defer this.propertyLock.Unlock()

	delete(this.propertyBag, key)
}
