package udpserv

import (
	//"fmt"

	"fmt"
	"io"
	"net"

	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"github.com/xtaci/kcp-go"

	//"strings"
	"sync"
	"time"
)

const g_udp_type = 1 //0: udp 1:kcp
const pool_size = 1500

var (
	pkgPool sync.Pool
	bufPool sync.Pool
)

func init() {
	pkgPool.New = func() interface{} {
		return &fnet.PkgAll{Pdata: &fnet.PkgData{}}
	}
	bufPool.New = func() interface{} {
		return make([]byte, pool_size)
	}
}

type UdpServ struct {
	sync.RWMutex
	dataChan     chan *DataReq
	sendChan     chan *DataReq
	exitChan     chan int
	exitSendChan chan int
	listener     *net.UDPConn
	kcplistener  *kcp.Listener
	datapack     iface.Idatapack
	Running      bool

	RecvChan  map[uint32]*chan *fnet.PkgAll
	CloseChan map[uint32]chan bool

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
		exitChan:     make(chan int, 1),
		exitSendChan: make(chan int, 1),
		CloseChan:    make(map[uint32]chan bool),
		datapack:     NewKCPDataPack(),
		RecvChan:     make(map[uint32]*chan *fnet.PkgAll),
		Running:      true,
	}
	fmt.Println("NewUdpServ!!!")
	GlobalUdpServ.StartKcpServ(port)
}

func (this *UdpServ) OperCloseCh(pid uint32) {
	this.Lock()
	defer this.Unlock()
	ch, ok := this.CloseChan[pid]
	if ok {
		ch <- false
	}
	delete(this.CloseChan, pid)
}

func (this *UdpServ) AddCloseCh(pid uint32, ch chan bool) {
	this.Lock()
	defer this.Unlock()
	this.CloseChan[pid] = ch
}

func (this *UdpServ) PushPkg(i interface{}) {
	pkg := i.(*fnet.PkgAll)
	slc := pkg.Pdata.Data
	if slc != nil {
		bufPool.Put(slc)
	}
	pkgPool.Put(pkg)
}

func (this *UdpServ) GetChRecv(roomid uint32) *chan *fnet.PkgAll {
	this.Lock()
	defer this.Unlock()
	val, ok := this.RecvChan[roomid]
	if !ok {
		ch := make(chan *fnet.PkgAll, 512)
		this.RecvChan[roomid] = &ch
		logger.Infof("init ch: %p roomid: %d", &ch, roomid)
		return &ch
	}
	return val
}
func (this *UdpServ) DelChRecv(roomid uint32) {
	this.Lock()
	defer this.Unlock()
	/*
		chp, ok := this.RecvChan[roomid]
		if ok {
			close(*chp)
		}
	*/
	delete(this.RecvChan, roomid)
}
func (this *UdpServ) UpdateChRecv(roomid uint32, chp *chan *fnet.PkgAll) {
	this.Lock()
	defer this.Unlock()
	this.RecvChan[roomid] = chp
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
		this.msgHandle = utils.GlobalObject.ProtocGate.GetMsgHandle()
	}
	bOver := false
	for {
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
		kcpidx := 0

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
				this.msgHandle = utils.GlobalObject.ProtocGate.GetMsgHandle()
			}
			kcpidx++

			go func(conn *kcp.UDPSession, idx int) {
				var chp *chan *fnet.PkgAll
				defer func() {
					if chp != nil {
						//close(*chp)
					}
				}()
				conn.SetReadBuffer(4 * 1024 * 1024)
				conn.SetWriteBuffer(4 * 1024 * 1024)
				logger.Infof("udpserv kcp new conn idx: %d", idx)
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
				roomid := uint32(0)
				head := make([]byte, this.datapack.GetHeadLen())
				bfirst := true

				chClose := make(chan bool, 12)
				isAdd := false
				for {
					select {
					case <-chClose:
						logger.Infof("udpserv goroutine exit: %d", idx)
						//this.kcplistener.Close()
						return
						//bOver = true
						//break
					case <-this.exitChan:
						logger.Infof("udpserv goroutine exit: %d", idx)
						this.kcplistener.Close()
						return
						//bOver = true
						//break
					default:
						if bfirst {
							conn.SetReadDeadline(time.Now().Add(2 * time.Second))
							bfirst = false
						} else {
							conn.SetReadDeadline(time.Now().Add(1 * 60 * time.Second))
						}
						_, err := io.ReadFull(conn, head)
						if err != nil {
							logger.Infof("kcpserv read head exit: %d", idx)
							conn.Close()
							return
						}
						pkgItf, err := this.datapack.Unpack(head, pkgPool.Get())
						if err != nil {
							logger.Infof("kcpserv read unpack exit: %d", idx)
							conn.Close()
							return
						}
						pkgAll := pkgItf.(*fnet.PkgAll)

						pid = uint32(pkgAll.Pid & 0xffffffff)
						if !isAdd {
							this.AddCloseCh(pid, chClose)
							isAdd = true
						}
						//for room id
						roomid = uint32(pkgAll.Pid >> 32)

						pkg := pkgAll.Pdata
						if pkg.Len > 0 {
							//pkg.Data = pkgAll.Data
							//pkg.Data = make([]byte, pkg.Len)
							//pkg.Data = tmpData[:pkg.Len]
							pkg.Data = bufPool.Get().([]byte)[:pkg.Len]
							_, err := conn.Read(pkg.Data)
							if err != nil {
								logger.Infof("kcpserv read data exit: %v msgid: %d len: %d", conn, pkg.MsgId, pkg.Len)
								conn.Close()
								return
							}
						}
						//this.msgHandle.UpdateNetIn(8 + 4 + int(pkg.Len))

						if utils.GlobalObject.UnmarshalPt != nil {
							utils.GlobalObject.UnmarshalPt(pkg)
						}

						/*
							obj := utils.GlobalObject.ProtocGate.GetMsgHandle()
								logger.Infof("kcpserv pid: %d read <%v> msgid: %d len: %d obj: %p name: %s",
									pid, conn.RemoteAddr(), pkg.MsgId, pkg.Len, obj, obj.Name())
						*/

						pkgAll.UdpConn = conn

						if chp == nil {
							chp = GlobalUdpServ.GetChRecv(roomid)
						}
						*chp <- pkgAll

						//logger.Infof("kcpserv kcpid: %d transfer msgid: %d len: %d roomid: %d ch: %p", idx, pkg.MsgId, pkg.Len, roomid, chp)
						//this.msgHandle.DeliverToMsgQueue(pkgAll)
					}
				}
			}(s.(*kcp.UDPSession), kcpidx)
		}
	}()
	go this.StartWriteThread()
}
