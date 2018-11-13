package cluster

import (
	//"fmt"
	"github.com/viphxin/xingo/iface"
	//"github.com/viphxin/xingo/logger"
	//"github.com/viphxin/xingo/utils"
	"time"
)

var REQUEST_NORESULT uint8 = uint8(0)
var REQUEST_FORRESULT uint8 = uint8(1)
var RESPONSE uint8 = uint8(2)
var REQUEST_SUCC uint8 = uint8(3)

type XingoRpc struct {
	conn           iface.IWriter
	asyncResultMgr *AsyncResultMgr
}

func NewXingoRpc(conn iface.IWriter) *XingoRpc {
	return &XingoRpc{
		conn:           conn,
		asyncResultMgr: AResultGlobalObj,
	}
}

func (this *XingoRpc) CallRpcNotForResult(target string, param string, pid uint64, msgid uint32, binData []byte) error {
	rpcdata := &RpcData{
		MsgType: REQUEST_NORESULT,
		Target:  target,
		Param:   param,
	}
	rpcdata.Bin = &RpcDataBin{
		Pid:     pid,
		Msgid:   msgid,
		BinData: binData,
	}
	this.conn.Send(rpcdata.Encode())
	return nil
}

func (this *XingoRpc) callRpcHasResp(target string, param string, pid uint32, msgid uint32, binData []byte, msgtype uint8) (*RpcData, error) {
	asyncR := this.asyncResultMgr.Add()
	rpcdata := &RpcData{
		MsgType: msgtype,
		Target:  target,
		Param:   param,
		Key:     asyncR.GetKey(),
	}
	rpcdata.Bin = &RpcDataBin{
		Pid:     uint64(pid),
		Msgid:   msgid,
		BinData: binData,
	}
	this.conn.Send(rpcdata.Encode())
	resp, err := asyncR.GetResult(5 * time.Second)
	if err == nil {
		return resp, nil
	} else {
		//超时了 或者其他原因结果没等到
		this.asyncResultMgr.Remove(asyncR.GetKey())
		return nil, err
	}
}

func (this *XingoRpc) CallRpcForResult(target string, param string, pid uint32, msgid uint32, binData []byte) (*RpcData, error) {
	return this.callRpcHasResp(target, param, pid, msgid, binData, REQUEST_FORRESULT)
}

func (this *XingoRpc) CallRpcSucc(target string, param string, pid uint32, msgid uint32, binData []byte) (*RpcData, error) {
	return this.callRpcHasResp(target, param, pid, msgid, binData, REQUEST_SUCC)
}
