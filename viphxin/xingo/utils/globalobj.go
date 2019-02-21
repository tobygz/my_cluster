package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/timer"
)

type GlobalObj struct {
	TcpServer              iface.Iserver
	Dbmgr                  iface.IDbop
	OnConnectioned         func(fconn iface.Iconnection)
	OnClosed               func(fconn iface.Iconnection)
	OnClusterConnectioned  func(fconn iface.Iconnection) //集群rpc root节点回调
	OnClusterClosed        func(fconn iface.Iconnection)
	OnClusterCConnectioned func(fconn iface.Iclient) //集群rpc 子节点回调
	OnClusterCClosed       func(fconn iface.Iclient)
	OnServerStop           func() //服务器停服回调
	OnServerStart          func() //服务器停服回调
	OnServerMsTimer        func() //服务器timer
	UnmarshalPt            func(interface{})
	ProtocGate             iface.IServerProtocol
	Protoc                 iface.IServerProtocol
	RpcSProtoc             iface.IServerProtocol
	RpcCProtoc             iface.IClientProtocol
	TcpPort                int
	MaxConn                int
	//log
	LogPath          string
	LogName          string
	BatDtlPath       string
	MaxLogNum        int32
	MaxFileSize      int64
	LogFileUnit      logger.UNIT
	LogLevel         logger.LEVEL
	SetToConsole     bool
	LogFileType      int32
	ToSyslog         bool
	SyslogAddr       string
	SyslogPort       int
	LogFileLine      bool
	PoolSize         int32
	IsUsePool        bool
	MaxWorkerLen     int32
	MaxSendChanLen   int32
	FrameSpeed       uint8
	Name             string
	MaxPacketSize    uint32
	FrequencyControl string //  100/h, 100/m, 100/s
	EnableFlowLog    bool
	MaxRid           uint64
	TimeChan         chan *timer.Timer
	KcpIs            bool
	UdpIp            string
	UdpPort          int
	WebObj           iface.Iweb
	IsClose          bool
	PProfAddr        string
	Encrypt          bool
}

func (this *GlobalObj) IncMaxRid() uint64 {
	this.MaxRid = this.MaxRid + 1
	return this.MaxRid
}

func (this *GlobalObj) GetFrequency() (int, string) {
	fc := strings.Split(this.FrequencyControl, "/")
	if len(fc) != 2 {
		return 0, ""
	} else {
		fc0_int, err := strconv.Atoi(fc[0])
		if err == nil {
			return fc0_int, fc[1]
		} else {
			logger.Error("FrequencyControl params error: ", this.FrequencyControl)
			return 0, ""
		}
	}
}

func (this *GlobalObj) IsWin() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

func (this *GlobalObj) IsGame() bool {
	return strings.Contains(this.Name, "game")
}

func (this *GlobalObj) IsGate() bool {
	return strings.Contains(this.Name, "gate")
}
func (this *GlobalObj) IsNet() bool {
	return strings.Contains(this.Name, "net")
}
func (this *GlobalObj) IsAdmin() bool {
	return strings.Contains(this.Name, "admin")
}

var GlobalObject *GlobalObj

func init() {
	GlobalObject = &GlobalObj{
		TcpPort:                8109,
		MaxConn:                12000,
		LogPath:                "./log",
		LogName:                "server.log",
		BatDtlPath:             "/data/batdtl",
		MaxLogNum:              10,
		MaxFileSize:            1024 * 1024 * 10,
		LogFileUnit:            logger.KB,
		LogLevel:               logger.ERROR,
		SetToConsole:           true,
		LogFileType:            2,
		ToSyslog:               false,
		LogFileLine:            true,
		PoolSize:               1,
		IsUsePool:              true,
		MaxWorkerLen:           1024,
		MaxSendChanLen:         1024,
		FrameSpeed:             30,
		EnableFlowLog:          false,
		IsClose:                false,
		Encrypt:                false,
		TimeChan:               make(chan *timer.Timer),
		OnConnectioned:         func(fconn iface.Iconnection) {},
		OnClosed:               func(fconn iface.Iconnection) {},
		OnClusterConnectioned:  func(fconn iface.Iconnection) {},
		OnClusterClosed:        func(fconn iface.Iconnection) {},
		OnClusterCConnectioned: func(fconn iface.Iclient) {},
		OnClusterCClosed:       func(fconn iface.Iclient) {},
	}

	//读取用户自定义配置
	data, err := ioutil.ReadFile("conf/server.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &GlobalObject)
	if err != nil {
		panic(err)
	}
}
