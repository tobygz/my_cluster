package cluster

import (
	"encoding/binary"

	"fmt"

	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/iface"
)

type RpcPackege struct {
	Len  int32
	Data []byte
}

type RpcRequest struct {
	Fconn   iface.IWriter
	Rpcdata *RpcData
}

func (this *RpcRequest) GetWriter() iface.IWriter {
	return this.Fconn
}

func (this *RpcRequest) GetMsgType() uint8 {
	return this.Rpcdata.MsgType
}

func (this *RpcRequest) GetKey() string {
	return this.Rpcdata.Key
}

func (this *RpcRequest) GetTarget() string {
	return this.Rpcdata.Target
}

func (this *RpcRequest) GetParam() string {
	return this.Rpcdata.Param
}

func (this *RpcRequest) GetResult() string {
	return this.Rpcdata.Result
}

func (this *RpcRequest) SetResult(result string) {
	this.Rpcdata.Result = result
}

func (this *RpcRequest) GetConnection() iface.Iconnection {
	return nil
}
func (this *RpcRequest) GetUdpConn() iface.IUdpConn {
	return nil
}
func (this *RpcRequest) GetData() []byte {
	return this.Rpcdata.Bin.BinData
}

func (this *RpcRequest) GetPid() uint64 {
	return this.Rpcdata.Bin.Pid
}

func (this *RpcRequest) GetMsgid() uint32 {
	return this.Rpcdata.Bin.Msgid
}

type RpcDataPack struct{}

func NewRpcDataPack() *RpcDataPack {
	return &RpcDataPack{}
}

func (this *RpcDataPack) GetHeadLen() int32 {
	return 4
}

func (this *RpcDataPack) Unpack(data []byte, pkgItf interface{}) (interface{}, error) {
	pkg, ok := pkgItf.(*RpcPackege)
	if !ok {
		pkg = &RpcPackege{}
	}

	if len(head) != int(this.GetHeadLen()) {
		return nil, fmt.Errorf("invalid head length")
	}

	// 读取Len
	pkg.Len = int32(binary.LittleEndian.Uint32(data))

	// 封包太大
	if pkg.Len > fnet.MaxPacketSize {
		return nil, fmt.Errorf("rpc package exceed: %d, rpc size: %d", fnet.MaxPacketSize, pkg.Len)
	}

	return pkg, nil
}

func (this *RpcDataPack) Pack(msgId uint32, pkg interface{}, b []byte) (o []byte, err error) {
	/*
	   gob.Register(RpcData{})
	   gob.Register([]interface{}{})
	   outbuff := bytes.NewBuffer([]byte{})
	   // 进行编码
	   databuff := bytes.NewBuffer([]byte{})
	   data := pkg.(*RpcData)
	   if data != nil {
	       enc := gob.NewEncoder(databuff)
	       err = enc.Encode(data)
	   }

	   if err != nil {
	       fmt.Println(fmt.Sprintf("gob marshaling error:  %s pkg: %v", err, data))
	   }
	   // 写Len
	   if err = binary.Write(outbuff, binary.LittleEndian, uint32(databuff.Len())); err != nil {
	       return
	   }

	   //all pkg data
	   if err = binary.Write(outbuff, binary.LittleEndian, databuff.Bytes()); err != nil {
	       return
	   }

	   out = outbuff.Bytes()
	*/
	return nil, nil
}
