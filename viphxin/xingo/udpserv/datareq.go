package udpserv

import (
	"net"

	"github.com/viphxin/xingo/iface"
	kcp "github.com/xtaci/kcp-go"
)

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
