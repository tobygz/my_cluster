package cluster

import (
	"bytes"
	"encoding/binary"
	"fmt"
	//"github.com/golang/protobuf/proto"
	"github.com/viphxin/xingo/utils"
)

type RpcData struct {
	MsgType uint8
	Target  string
	Key     string
	Param   string
	Result  string
	Bin     *RpcDataBin
}

type RpcDataBin struct {
	/*
	   1. rpcname
	   2. pid
	   3. msgid
	   4. []byte
	   5. len
	*/
	Pid     uint64
	Msgid   uint32
	BinData []byte
}

func (this *RpcData) String() string {
	return fmt.Sprintf("msgtype: %d key: %s param: %s target: %s bin.pid: %d msgid: %d bindata: %s", this.MsgType,
		this.Key, this.Param, this.Target, this.Bin.Pid, this.Bin.Msgid, string(this.Bin.BinData))
}

func (this *RpcData) Size() uint32 {
	sz := uint32(1)     //msgtype
	sz = sz + uint32(1) //key len
	sz = sz + uint32(len(this.Key))
	sz = sz + uint32(1) //target len
	sz = sz + uint32(len(this.Target))

	sz = sz + uint32(1) //param len
	sz = sz + uint32(len(this.Param))

	sz = sz + uint32(1) //result len
	sz = sz + uint32(len(this.Result))

	if this.Bin != nil {
		sz = sz + uint32(8)                     // bin.pid
		sz = sz + uint32(4)                     //bin.msgid
		sz = sz + uint32(4)                     //bin len
		sz = sz + uint32(len(this.Bin.BinData)) //bin
	}
	return sz
}

func (this *RpcData) Encode() []byte {
	outbuff := bytes.NewBuffer([]byte{})

	if err := binary.Write(outbuff, binary.LittleEndian, this.Size()); err != nil {
		panic(err)
	}
	//msgtype

	if err := binary.Write(outbuff, binary.LittleEndian, this.MsgType); err != nil {
		panic(err)
	}
	//key
	if err := binary.Write(outbuff, binary.LittleEndian, uint8(len(this.Key))); err != nil {
		panic(err)
	}
	if err := binary.Write(outbuff, binary.LittleEndian, utils.Str2bytes(this.Key)); err != nil {
		panic(err)
	}
	//target
	if err := binary.Write(outbuff, binary.LittleEndian, uint8(len(this.Target))); err != nil {
		panic(err)
	}
	if err := binary.Write(outbuff, binary.LittleEndian, utils.Str2bytes(this.Target)); err != nil {
		panic(err)
	}

	//param
	if err := binary.Write(outbuff, binary.LittleEndian, uint8(len(this.Param))); err != nil {
		panic(err)
	}
	if err := binary.Write(outbuff, binary.LittleEndian, utils.Str2bytes(this.Param)); err != nil {
		panic(err)
	}

	//result
	if err := binary.Write(outbuff, binary.LittleEndian, uint8(len(this.Result))); err != nil {
		panic(err)
	}
	if err := binary.Write(outbuff, binary.LittleEndian, []byte(this.Result)); err != nil {
		panic(err)
	}

	if this.Bin == nil {
		return outbuff.Bytes()
	}

	//bin.pid
	if err := binary.Write(outbuff, binary.LittleEndian, this.Bin.Pid); err != nil {
		panic(err)
	}
	//bin.Msgid
	if err := binary.Write(outbuff, binary.LittleEndian, this.Bin.Msgid); err != nil {
		panic(err)
	}

	binLen := uint32(0)
	if this.Bin.BinData != nil {
		binLen = uint32(len(this.Bin.BinData))
	}
	if err := binary.Write(outbuff, binary.LittleEndian, binLen); err != nil {
		panic(err)
	}
	if binLen != 0 {
		if err := binary.Write(outbuff, binary.LittleEndian, this.Bin.BinData); err != nil {
			panic(err)
		}
	}
	return outbuff.Bytes()
}

func (this *RpcData) Decode(data []byte) {
	outbuff := bytes.NewBuffer(data)

	if err := binary.Read(outbuff, binary.LittleEndian, &this.MsgType); err != nil {
		panic(err)
		return
	}
	//key
	lenv := uint8(0)
	if err := binary.Read(outbuff, binary.LittleEndian, &lenv); err != nil {
		panic(err)
		return
	}
	keySlc := make([]byte, lenv)
	if err := binary.Read(outbuff, binary.LittleEndian, &keySlc); err != nil {
		panic(err)
		return
	}
	this.Key = utils.Bytes2str(keySlc)

	//target
	if err := binary.Read(outbuff, binary.LittleEndian, &lenv); err != nil {
		panic(err)
		return
	}
	tarSlc := make([]byte, lenv)
	if err := binary.Read(outbuff, binary.LittleEndian, &tarSlc); err != nil {
		panic(err)
		return
	}
	this.Target = utils.Bytes2str(tarSlc)

	//param
	if err := binary.Read(outbuff, binary.LittleEndian, &lenv); err != nil {
		panic(err)
		return
	}
	paramSlc := make([]byte, lenv)
	if err := binary.Read(outbuff, binary.LittleEndian, &paramSlc); err != nil {
		panic(err)
		return
	}
	this.Param = utils.Bytes2str(paramSlc)

	//result
	if err := binary.Read(outbuff, binary.LittleEndian, &lenv); err != nil {
		panic(err)
		return
	}
	resSlc := make([]byte, lenv)
	if err := binary.Read(outbuff, binary.LittleEndian, &resSlc); err != nil {
		panic(err)
		return
	}
	this.Result = utils.Bytes2str(resSlc)

	if outbuff.Len() == 0 {
		return
	}

	if this.Bin == nil {
		this.Bin = &RpcDataBin{}
	}
	//bin.pid
	if err := binary.Read(outbuff, binary.LittleEndian, &this.Bin.Pid); err != nil {
		panic(err)
		return
	}
	//bin.Msgid
	if err := binary.Read(outbuff, binary.LittleEndian, &this.Bin.Msgid); err != nil {
		panic(err)
		return
	}

	binLen := uint32(0)
	if err := binary.Read(outbuff, binary.LittleEndian, &binLen); err != nil {
		panic(err)
		return
	}

	this.Bin.BinData = make([]byte, binLen)
	if err := binary.Read(outbuff, binary.LittleEndian, &this.Bin.BinData); err != nil {
		panic(err)
		return
	}

}
