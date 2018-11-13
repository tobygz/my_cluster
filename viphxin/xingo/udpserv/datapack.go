package udpserv

import (
	"encoding/binary"
	"fmt"

	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/utils"
)

type KCPDataPack struct{}

func NewKCPDataPack() *KCPDataPack {
	obj := &KCPDataPack{}
	return obj
}

func (d *KCPDataPack) GetHeadLen() int32 {
	return 16
}

func (d *KCPDataPack) Unpack(head []byte, pkgItf interface{}) (interface{}, error) {
	pkg, ok := pkgItf.(*fnet.PkgAll)
	if !ok {
		pkg = &fnet.PkgAll{Pdata: &fnet.PkgData{}}
	}
	if pkg.Pdata == nil {
		pkg.Pdata = &fnet.PkgData{}
	}

	pkg.Pdata.Len = binary.LittleEndian.Uint32(head[0:])
	pkg.Pdata.MsgId = binary.LittleEndian.Uint32(head[4:])
	pkg.Pid = binary.LittleEndian.Uint64(head[8:])

	if utils.GlobalObject.MaxPacketSize > 0 {
		if pkg.Pdata.Len > utils.GlobalObject.MaxPacketSize {
			return nil, fmt.Errorf("msgId: %d exceed: %d, msg size: %d", pkg.Pdata.MsgId, utils.GlobalObject.MaxPacketSize, pkg.Pdata.Len)
		}
	} else if pkg.Pdata.Len > fnet.MaxPacketSize {
		return nil, fmt.Errorf("msgId: %d exceed: %d, msg size: %d", pkg.Pdata.MsgId, fnet.MaxPacketSize, pkg.Pdata.Len)
	}

	return pkg, nil
}

func (d *KCPDataPack) Pack(msgId uint32, data interface{}, b []byte) ([]byte, error) {
	return nil, nil
}
