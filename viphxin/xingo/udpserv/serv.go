package udpserv

import (
	//"fmt"
	"encoding/binary"
	"fmt"
	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"github.com/xtaci/kcp-go"
	"net"
	"strings"
	"sync"
	"time"
)

const g_udp_type = 1 //0: udp 1:kcp

type DataReq struct {
	addr    *net.UDPAddr
	kcpconn *kcp.UDPSession
	data    []byte
	pid     uint32
}

func (this *DataReq) GetAddr() *net.UDPAddr {
	return this.addr
}

func (this *DataReq) GetKcp() *kcp.UDPSession {
	return this.kcpconn
}

func (this *DataReq) GetConnection() iface.Iconnection {
	return nil
}

func (this *DataReq) GetData() []byte {
	return this.data
}

func (this *DataReq) GetMsgId() uint32 {
	return 0
}

type UdpServ struct {
	sync.RWMutex
	dataChan     chan *DataReq
	sendChan     chan *DataReq
	exitChan     chan int
	exitSendChan chan int
	listener     *net.UDPConn
	kcplistener  *kcp.Listener
	pbdataPack   *fnet.PBDataPack
	Running      bool

	msgHandle iface.Imsghandle
}

var GlobalUdpServ *UdpServ = nil

func NewUdpServ(port int, kcpEnable bool) {
	if GlobalUdpServ != nil {
		return
	}
	GlobalUdpServ = &UdpServ{
		dataChan:     make(chan *DataReq, 32),
		sendChan:     make(chan *DataReq, 1024),
		exitChan:     make(chan int, 10),
		exitSendChan: make(chan int, 10),
		pbdataPack:   fnet.NewPBDataPack(),
		Running:      true,
	}
	if !kcpEnable {
		GlobalUdpServ.StartServ(port)
	} else {
		GlobalUdpServ.StartKcpServ(port)
	}
}

func (this *UdpServ) Close() {
	this.Running = false
	this.exitChan <- 0
	this.exitChan <- 0
	this.exitSendChan <- 0
	this.exitSendChan <- 0
}

func (this *UdpServ) GetChan() chan *DataReq {
	return this.dataChan
}

func (this *UdpServ) Send(addr *net.UDPAddr, conn *kcp.UDPSession, dataBt []byte, pid uint32) {
	if conn.IsClosed() {
		return
	}
	st := &DataReq{
		addr:    addr,
		kcpconn: conn,
		data:    dataBt,
		pid:     pid,
	}
	this.sendChan <- st
}

func (this *UdpServ) StartWriteThread() {
	if this.msgHandle == nil {
		this.msgHandle = utils.GlobalObject.Protoc.GetMsgHandle()
	}
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
			if st.kcpconn != nil {
				_, err := st.kcpconn.Write(st.data)
				this.msgHandle.UpdateNetOut(len(st.data))
				if err != nil {
					logger.Error("udpserv err: %s", err.Error())
				}
			}
		}
	}
}

func listenEcho(port string) (net.Listener, error) {
	return kcp.ListenWithOptions(port, nil, 10, 3)
}

func (this *UdpServ) StartKcpServ(port int) {
	go func() {
		ports := fmt.Sprintf("0.0.0.0:%d", port)
		l, err := listenEcho(ports)
		if err != nil {
			panic(err)
			return
		}
		this.kcplistener = l.(*kcp.Listener)
		this.kcplistener.SetReadBuffer(4 * 1024 * 1024)
		this.kcplistener.SetWriteBuffer(4 * 1024 * 1024)
		this.kcplistener.SetDSCP(46)

		logger.Info("udpserv kcp listen: ", ports)
		//data := make([]byte, 5)
		bOver := false
		for {
			if bOver {
				close(this.dataChan)
				close(this.exitChan)
				break
			}
			s, err := this.kcplistener.Accept()
			if err != nil {
				logger.Error("!!!kcplistener.Accept err:", err)
				return
			}
			if this.msgHandle == nil {
				this.msgHandle = utils.GlobalObject.Protoc.GetMsgHandle()
			}

			go func(conn *kcp.UDPSession) {
				conn.SetReadBuffer(4 * 1024 * 1024)
				conn.SetWriteBuffer(4 * 1024 * 1024)
				logger.Info("udpserv kcp new conn: ")
				conn.SetStreamMode(true)
				conn.SetWindowSize(4096, 4096)
				conn.SetNoDelay(1, 10, 2, 1)
				conn.SetDSCP(46)
				conn.SetMtu(1400)
				conn.SetACKNoDelay(false)
				conn.SetReadDeadline(time.Now().Add(time.Hour))
				conn.SetWriteDeadline(time.Now().Add(time.Hour))
				//rid := uint64(0)
				pid := uint32(0)
				pidary := make([]byte, 4)
				bfirst := true

				ticker := time.NewTicker(1 * time.Millisecond)
				for {
					select {
					case <-this.exitChan:
						logger.Info("udpserv goroutine exit")
						this.kcplistener.Close()
						bOver = true
						break
					case <-ticker.C:

						head := make([]byte, (*this.pbdataPack).GetHeadLen())
						if bfirst {
							conn.SetReadDeadline(time.Now().Add(2 * time.Second))
							bfirst = false
						} else {
							conn.SetReadDeadline(time.Now().Add(1 * 60 * time.Second))
						}
						_, err := conn.Read(head)
						if err != nil {
							logger.Infof("kcpserv read head exit : %v", conn)
							conn.Close()
							return
						}
						pkgHead, err := (*this.pbdataPack).Unpack(head)
						if err != nil {
							logger.Infof("kcpserv read unpack exit : headdata: conn: %v err: %v", conn, err)
							conn.Close()
							return
						}
						_, err = conn.Read(pidary)
						if err != nil {
							logger.Infof("kcpserv read pidary exit : %v", conn)
							conn.Close()
							return
						}
						pid = binary.LittleEndian.Uint32(pidary)
						pkg := pkgHead.(*fnet.PkgData)
						if pkg.Len > 0 {
							pkg.Data = make([]byte, pkg.Len)
							_, err := conn.Read(pkg.Data)
							this.msgHandle.UpdateNetIn(8 + 4 + int(pkg.Len))
							if err != nil {
								logger.Infof("kcpserv read data exit : %v msgid: %d len: %d", conn, pkg.MsgId, pkg.Len)
								conn.Close()
								return
							}
						}

						/*
							obj := utils.GlobalObject.ProtocGate.GetMsgHandle()
								logger.Infof("kcpserv pid: %d read <%v> msgid: %d len: %d obj: %p name: %s",
									pid, conn.RemoteAddr(), pkg.MsgId, pkg.Len, obj, obj.Name())
						*/
						pkgAll := &fnet.PkgAll{
							Pdata:   pkg,
							Pid:     uint64(pid),
							Fconn:   nil,
							UdpConn: conn,
						}
						utils.GlobalObject.ProtocGate.GetMsgHandle().DeliverToMsgQueue(pkgAll)
					}
				}
			}(s.(*kcp.UDPSession))
		}
	}()
	go this.StartWriteThread()
}

func (this *UdpServ) StartServ(port int) {
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
					addr:    remoteAddr,
					kcpconn: nil,
					data:    data[:n],
				}
				this.dataChan <- st
			}
		}
	}()
}
