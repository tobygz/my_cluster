package udpserv

import (
	//"fmt"
	"github.com/viphxin/xingo/logger"
	"net"
	"strings"
	"time"
)

type DataReq struct {
	addr *net.UDPAddr
	data []byte
}

func (this *DataReq) GetAddr() *net.UDPAddr {
	return this.addr
}

func (this *DataReq) GetData() []byte {
	return this.data
}

type UdpServ struct {
	dataChan     chan *DataReq
	sendChan     chan *DataReq
	exitChan     chan int
	exitSendChan chan int
	listener     *net.UDPConn
	Running      bool
}

var GlobalUdpServ *UdpServ = nil

func NewUdpServ(port int) {
	GlobalUdpServ = &UdpServ{
		dataChan:     make(chan *DataReq, 32),
		sendChan:     make(chan *DataReq, 1024),
		exitChan:     make(chan int),
		exitSendChan: make(chan int),
		Running:      true,
	}
	GlobalUdpServ.Start(port)
}

func (this *UdpServ) Close() {
	this.Running = false
	this.exitChan <- 0
	this.exitSendChan <- 0
}

func (this *UdpServ) GetChan() chan *DataReq {
	return this.dataChan
}

func (this *UdpServ) Send(addr *net.UDPAddr, dataBt []byte) {
	st := &DataReq{
		addr: addr,
		data: dataBt,
	}
	this.sendChan <- st
}

func (this *UdpServ) Start(port int) {
	go func() {
		for {
			bOver := false
			if bOver {
				close(this.exitSendChan)
				close(this.sendChan)
				break
			}

			select {
			case <-this.exitSendChan:
				bOver = true
			case st := <-this.sendChan:
				_, err := this.listener.WriteToUDP(st.data, st.addr)
				if err != nil {
					logger.Error("udpserv err: %s", err.Error())
				}
			}
		}
	}()
	go func() {
		var err error
		this.listener, err = net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port})
		if err != nil {
			panic(err)
			return
		}
		logger.Info("udpserv listen: %s", this.listener.LocalAddr().String())
		data := make([]byte, 1024)
		bOver := false
		for {
			if bOver {
				close(this.dataChan)
				close(this.exitChan)
				break
			}
			select {
			case <-this.exitChan:
				logger.Info("udpserv goroutine exit")
				bOver = true
				break
			case <-time.After(0):
				this.listener.SetReadDeadline(time.Now().Add(time.Millisecond * 500))
				n, remoteAddr, err := this.listener.ReadFromUDP(data)
				if err != nil {
					if strings.Contains(err.Error(), "timeout") {
						continue
					}
					logger.Error("error during read: %s", err)
				}
				//fmt.Printf("<%s> %s\n", remoteAddr, data[:n])
				st := &DataReq{
					addr: remoteAddr,
					data: data[:n],
				}
				this.dataChan <- st
			}
		}
	}()
}
