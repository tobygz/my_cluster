package fnet

import (
	"encoding/binary"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/utils"
)

type PkgData struct {
	Len   uint32
	MsgId uint32
	Data  []byte
	PbObj interface{}
}

func (this *PkgData) SetMsgObj(inter interface{}) {
	this.PbObj = inter
}

type PBDataPack struct {
	bNet      bool
	msgHandle iface.Imsghandle
}

func NewPBDataPack() *PBDataPack {
	obj := &PBDataPack{}
	obj.bNet = utils.GlobalObject.IsNet()
	return obj
}

func (this *PBDataPack) GetHeadLen() int32 {
	return 8
}

func (this *PBDataPack) Unpack(head []byte, pkgItf interface{}) (interface{}, error) {
	pkg, ok := pkgItf.(*PkgData)
	if !ok {
		pkg = &PkgData{}
	}

	if len(head) != int(this.GetHeadLen()) {
		return nil, fmt.Errorf("invalid head length")
	}

	// 读取Len
	pkg.Len = binary.LittleEndian.Uint32(head[0:])
	// 读取MsgId
	pkg.MsgId = binary.LittleEndian.Uint32(head[4:])

	if this.bNet {
		if this.msgHandle == nil {
			this.msgHandle = utils.GlobalObject.Protoc.GetMsgHandle()
		}
		this.msgHandle.UpdateNetIn(int(this.GetHeadLen()) + int(pkg.Len))
	}

	// 封包太大
	if utils.GlobalObject.MaxPacketSize > 0 {
		if pkg.Len > utils.GlobalObject.MaxPacketSize {
			return nil, packageTooBig
		}
	} else if pkg.Len > MaxPacketSize {
		return nil, packageTooBig
	}

	return pkg, nil
}

func (this *PBDataPack) Pack(msgId uint32, pkg interface{}, b []byte) (o []byte, err error) {
	// 进行编码
	var bt []byte
	if pkg != nil {
		bt, err = proto.Marshal(pkg.(proto.Message))
		if err != nil {
			panic(err)
			//logger.Errorf("marshaling error:  %s", err)
		}
	}

	if cap(b) >= len(bt)+8 {
		o = b[0 : len(bt)+8]
	} else {
		o = make([]byte, len(bt)+8)
	}

	// 写Len
	binary.LittleEndian.PutUint32(o[0:], uint32(len(bt)))
	// 写MsgId
	binary.LittleEndian.PutUint32(o[4:], msgId)

	//all pkg data
	copy(o[8:], bt)

	return
}
