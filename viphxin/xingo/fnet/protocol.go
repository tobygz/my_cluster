package fnet

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/viphxin/xingo/encry"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
)

const (
	MaxPacketSize = 1024 * 1024
)

var (
	packageTooBig = errors.New("Too many data to received!!")
)

type PkgAll struct {
	Pdata   *PkgData
	Fconn   iface.Iconnection
	UdpConn iface.IUdpConn
	Pid     uint64
}

func (this *PkgAll) GetConnection() iface.Iconnection {
	return this.Fconn
}

func (this *PkgAll) GetUdpConn() iface.IUdpConn {
	return this.UdpConn
}

func (this *PkgAll) GetData() []byte {
	return this.Pdata.Data
}

func (this *PkgAll) GetMsgId() uint32 {
	return this.Pdata.MsgId
}

func (this *PkgAll) SetMsgObj(inter interface{}) {
	this.Pdata.SetMsgObj(inter)
}
func (this *PkgAll) GetMsgObj() interface{} {
	return this.Pdata.PbObj
}

type Protocol struct {
	msghandle  *MsgHandle
	pbdatapack *PBDataPack
	rc4        *encry.Cipher
	benc       bool
}

func NewProtocol() *Protocol {
	return &Protocol{
		msghandle:  NewMsgHandle(),
		pbdatapack: NewPBDataPack(),
		benc:       false,
	}
}

func (this *Protocol) InitRc4(key []byte) {
	if key == nil || len(key) == 0 {
		return
	}
	this.benc = true
	this.rc4, _ = encry.NewCipher(key)
}

func (this *Protocol) ManualMsgPush(msgId uint32, data []byte, pid uint64, fconn iface.Iconnection) {
	pData := &PkgData{
		Len:   uint32(len(data)),
		MsgId: msgId,
		Data:  data,
	}
	pkgAll := &PkgAll{
		Pdata: pData,
		Pid:   pid,
		Fconn: fconn,
	}

	if utils.GlobalObject.UnmarshalPt != nil {
		utils.GlobalObject.UnmarshalPt(pData)
	}

	this.msghandle.DeliverToMsgQueue(pkgAll)
}

func (this *Protocol) GetMsgHandle() iface.Imsghandle {
	return this.msghandle
}
func (this *Protocol) GetDataPack() iface.Idatapack {
	return this.pbdatapack
}

func (this *Protocol) AddRpcRouter(router interface{}) {
	this.msghandle.AddRouter(router.(iface.IRouter))
}

func (this *Protocol) InitWorker(poolsize int32) {
	this.msghandle.StartWorkerLoop(int(poolsize))
}

func (this *Protocol) OnConnectionMade(fconn iface.Iconnection) {
	logger.Info(fmt.Sprintf("client ID: %d connected. IP Address: %s", fconn.GetSessionId(), fconn.RemoteAddr()))
	utils.GlobalObject.OnConnectioned(fconn)
	//加频率控制
	//this.SetFrequencyControl(fconn)
}

func (this *Protocol) SetFrequencyControl(fconn iface.Iconnection) {
	fc0, fc1 := utils.GlobalObject.GetFrequency()
	if fc1 == "h" {
		fconn.SetProperty("xingo_fc", 0)
		fconn.SetProperty("xingo_fc0", fc0)
		fconn.SetProperty("xingo_fc1", time.Now().UnixNano()*1e6+int64(3600*1e3))
	} else if fc1 == "m" {
		fconn.SetProperty("xingo_fc", 0)
		fconn.SetProperty("xingo_fc0", fc0)
		fconn.SetProperty("xingo_fc1", time.Now().UnixNano()*1e6+int64(60*1e3))
	} else if fc1 == "s" {
		fconn.SetProperty("xingo_fc", 0)
		fconn.SetProperty("xingo_fc0", fc0)
		fconn.SetProperty("xingo_fc1", time.Now().UnixNano()*1e6+int64(1e3))
	}
}

func (this *Protocol) DoFrequencyControl(fconn iface.Iconnection) error {
	xingo_fc1, err := fconn.GetProperty("xingo_fc1")
	if err != nil {
		//没有频率控制
		return nil
	} else {
		if time.Now().UnixNano()*1e6 >= xingo_fc1.(int64) {
			//init
			//this.SetFrequencyControl(fconn)
		} else {
			xingo_fc, _ := fconn.GetProperty("xingo_fc")
			xingo_fc0, _ := fconn.GetProperty("xingo_fc0")
			xingo_fc_int := xingo_fc.(int) + 1
			xingo_fc0_int := xingo_fc0.(int)
			if xingo_fc_int >= xingo_fc0_int {
				//trigger
				return errors.New(fmt.Sprintf("received package exceed limit: %s", utils.GlobalObject.FrequencyControl))
			} else {
				fconn.SetProperty("xingo_fc", xingo_fc_int)
			}
		}
		return nil
	}
}

func (this *Protocol) OnConnectionLost(fconn iface.Iconnection) {
	logger.Info(fmt.Sprintf("client ID: %d disconnected. IP Address: %s", fconn.GetSessionId(), fconn.RemoteAddr()))
	utils.GlobalObject.OnClosed(fconn)
}

func (this *Protocol) StartReadThread(fconn iface.Iconnection) {
	logger.Info("start receive data from socket...")
	for {
		//频率控制
		err := this.DoFrequencyControl(fconn)
		if err != nil {
			logger.Error(err)
			fconn.Stop()
			return
		}
		//read per head data
		headdata := make([]byte, this.pbdatapack.GetHeadLen())

		if _, err := io.ReadFull(fconn.GetConnection(), headdata); err != nil {
			logger.Error(err)
			fconn.Stop()
			return
		}
		pkgHead, err := this.pbdatapack.Unpack(headdata, nil)
		if err != nil {
			logger.Error(err)
			fconn.Stop()
			return
		}
		//data
		pkg := pkgHead.(*PkgData)
		if pkg.Len > 0 {
			pkg.Data = make([]byte, pkg.Len)
			if _, err := io.ReadFull(fconn.GetConnection(), pkg.Data); err != nil {
				logger.Error(err)
				fconn.Stop()
				return
			}
		}
		logger.Infof("pkg len:", pkg.Len, " enc data: ", pkg.Data, " benc:", this.benc)
		if this.benc {
			dt := make([]byte, len(pkg.Data))
			this.rc4.XorKeyStreamGeneric(dt[0:], pkg.Data[0:])
			copy(pkg.Data[0:], dt)
		}

		logger.Debug(fmt.Sprintf("msg id :%d, data len: %d data: %v", pkg.MsgId, pkg.Len, pkg.Data))
		if utils.GlobalObject.IsUsePool {
			this.msghandle.DeliverToMsgQueue(&PkgAll{
				Pdata: pkg,
				Fconn: fconn,
			})
		} else {
			this.msghandle.DoMsgFromGoRoutine(&PkgAll{
				Pdata: pkg,
				Fconn: fconn,
			})
		}

	}
}
