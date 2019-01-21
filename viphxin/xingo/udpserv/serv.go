package udpserv

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	kcp "github.com/xtaci/kcp-go"
)

const (
	g_udp_type = 1 //0=udp,1=kcp TODO:fix udp mode
	pool_size  = 1024
)

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
	ln        net.Listener
	conns     map[uint32]net.Conn
	datapack  iface.Idatapack
	msghandle iface.Imsghandle

	sendChan     chan *DataReq
	recvChan     map[uint32]*chan *fnet.PkgAll
	exitSendChan chan int

	connMu  sync.Mutex
	chMu    sync.Mutex
	wgLn    sync.WaitGroup
	wgConns sync.WaitGroup
}

var GlobalUdpServ *UdpServ = nil

func NewUdpServ(port int) {
	if GlobalUdpServ != nil {
		return
	}

	GlobalUdpServ = &UdpServ{
		conns:    make(map[uint32]net.Conn, 1024),
		datapack: NewKCPDataPack(),

		sendChan: make(chan *DataReq, 1024),
		recvChan: make(map[uint32]*chan *fnet.PkgAll, 1024),
	}

	println("NewUdpServ!!!")
	GlobalUdpServ.Start(port)
}

func (this *UdpServ) Start(port int) {
	this.init(port)
	go this.run()
}

func (this *UdpServ) init(port int) {
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	l, err := kcp.ListenWithOptions(addr, nil, 7, 3) //dataShards=7,parityShards=3,for lost msg resend
	if err != nil {
		panic(err)
		return
	}

	l.SetReadBuffer(4 * 1024 * 1024)
	l.SetWriteBuffer(4 * 1024 * 1024)
	l.SetDSCP(46)
	this.ln = l

	if this.msghandle == nil {
		this.msghandle = utils.GlobalObject.ProtocGate.GetMsgHandle()
	}

	logger.Infof("UdpServ.init, listen: %s", addr)
}

func (this *UdpServ) run() {
	this.wgLn.Add(1)
	defer this.wgLn.Done()

	go this.writeThread()

	kcpIdx := 0
	for {
		c, err := this.ln.Accept()
		if err != nil {
			logger.Errorf("!!! kcpListener.Accept fail, err: %s", err)
			return
		}

		kcpIdx++
		this.wgConns.Add(1)
		go this.handleConn(c, kcpIdx)
	}
}

func (this *UdpServ) writeThread() {
	for {
		select {
		case <-this.exitSendChan:
			close(this.sendChan)
			return
		case st := <-this.sendChan:
			if st.kcpconn != nil {
				_, err := st.kcpconn.Write(st.data)
				this.msghandle.UpdateNetOut(len(st.data))
				if err != nil {
					logger.Errorf("UdpServ.writeThread, UDPSession.Write fail, err: %s", err)
				}
			}
		} //select
	} //for
}

func (this *UdpServ) handleConn(c net.Conn, kcpIdx int) {
	defer this.wgConns.Done()

	conn, ok := c.(*kcp.UDPSession)
	if !ok {
		logger.Errorf("UdpServ.handleConn fail, conn not kcp.UDPSession, idx %d", kcpIdx)
		return
	}

	//set by listener
	//conn.SetReadBuffer(4 * 1024 * 1024)
	//conn.SetWriteBuffer(4 * 1024 * 1024)
	//conn.SetDSCP(46)
	//set by conn
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Second * 2))
	conn.SetWriteDeadline(time.Now().Add(time.Hour)) //conn must close after one hour

	logger.Infof("UdpServ.handleConn, new conn idx: %d", kcpIdx)

	var pid, roomid uint32
	added := false
	headData := make([]byte, this.datapack.GetHeadLen())
	var chp *chan *fnet.PkgAll
	for {
		//protocol: { head 16[ length 4 ][ msgid 4 ][ sid 8( pid 4 )( roomid 4 ) ] }{ data length }
		_, err := io.ReadFull(conn, headData)
		if err != nil {
			logger.Errorf("UdpServ.handleConn, kcpIdx %d, read head err: %s", kcpIdx, err)
			this.doCloseConn(pid, c)
			return
		}

		pkgItf, err := this.datapack.Unpack(headData, pkgPool.Get())
		if err != nil {
			logger.Errorf("UdpServ.handleConn, kcpIdx %d, unpack err: %s", kcpIdx, err)
			this.doCloseConn(pid, c)
			return
		}
		pkgAll := pkgItf.(*fnet.PkgAll)

		pid = uint32(pkgAll.Pid & 0xffffffff)
		if !added {
			this.addConn(pid, c)
			added = true
		}

		pkg := pkgAll.Pdata
		if pkg.Len > 0 {
			pkg.Data = bufPool.Get().([]byte)[:pkg.Len]
			if _, err = io.ReadFull(conn, pkg.Data); err != nil {
				logger.Errorf("UdpServ.handleConn, kcpIdx %d, read data err: %s", kcpIdx, err)
				this.doCloseConn(pid, c)
				return
			}
		} else if pkg.Len == 0 && pkg.MsgId == 2000 {
			logger.Infof("drop msg2000 with len 0")
			continue
		}
		utils.GlobalObject.UnmarshalPt(pkg)
		pkgAll.UdpConn = conn

		roomid = uint32(pkgAll.Pid >> 32)
		if chp == nil {
			chp = this.GetChRecv(roomid)
		}
		*chp <- pkgAll

		conn.SetReadDeadline(time.Now().Add(time.Minute))
	}
}

func (this *UdpServ) doCloseConn(pid uint32, c net.Conn) {
	c.Close()

	this.connMu.Lock()
	defer this.connMu.Unlock()
	if _, ok := this.conns[pid]; ok {
		delete(this.conns, pid)
	}
}

func (this *UdpServ) addConn(pid uint32, c net.Conn) {
	this.connMu.Lock()
	defer this.connMu.Unlock()
	if conn, ok := this.conns[pid]; ok {
		conn.Close()
	}
	this.conns[pid] = c
}

func (this *UdpServ) CloseConn(pid uint32) {
	this.connMu.Lock()
	defer this.connMu.Unlock()
	if c, ok := this.conns[pid]; ok {
		c.Close()
		delete(this.conns, pid)
	}
}

func (this *UdpServ) PushPkg(pkgItf interface{}) {
	pkg, ok := pkgItf.(*fnet.PkgAll)
	if ok {
		if pkg.Pdata.Data != nil {
			bufPool.Put(pkg.Pdata.Data)
		}
		pkgPool.Put(pkg)
	}
}

func (this *UdpServ) GetChRecv(roomid uint32) *chan *fnet.PkgAll {
	this.chMu.Lock()
	defer this.chMu.Unlock()

	if chp, ok := this.recvChan[roomid]; ok {
		return chp
	} else {
		ch := make(chan *fnet.PkgAll, 512)
		this.recvChan[roomid] = &ch
		logger.Infof("UdpServ.GetChRecv, init ch: %p roomid: %d", &ch, roomid)
		return &ch
	}
}

func (this *UdpServ) DelChRecv(roomid uint32) {
	this.chMu.Lock()
	defer this.chMu.Unlock()

	delete(this.recvChan, roomid)
}

func (this *UdpServ) UpdateChRecv(roomid uint32, chp *chan *fnet.PkgAll) {
	this.chMu.Lock()
	defer this.chMu.Unlock()

	this.recvChan[roomid] = chp
}

func (this *UdpServ) Close() {
	this.ln.Close()
	this.wgLn.Wait()

	this.chMu.Lock()
	for _, c := range this.conns {
		c.Close()
	}
	this.conns = nil
	this.chMu.Unlock()

	this.wgConns.Wait()

	close(this.exitSendChan)
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
