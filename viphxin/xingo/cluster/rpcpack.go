package cluster

import (
	"bytes"
	"encoding/binary"
	//"encoding/gob"
	"errors"
	//"fmt"
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

func (this *RpcRequest) GetData() []byte {
	return this.Rpcdata.Bin.BinData
}

func (this *RpcRequest) GetPid() uint32 {
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

func (this *RpcDataPack) Unpack(headdata []byte) (interface{}, error) {
	headbuf := bytes.NewReader(headdata)

	rp := &RpcPackege{}

	// 读取Len
	if err := binary.Read(headbuf, binary.LittleEndian, &rp.Len); err != nil {
		return nil, err
	}

	// 封包太大
	if rp.Len > fnet.MaxPacketSize {
		return nil, errors.New("rpc packege too big!!!")
	}

	return rp, nil
}

func (this *RpcDataPack) Pack(msgId uint32, pkg interface{}) (out []byte, err error) {
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
