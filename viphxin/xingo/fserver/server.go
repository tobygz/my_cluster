package fserver

import (
	"fmt"
	"net"
	"os"
	"os/signal"

	"github.com/viphxin/xingo/fnet"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/timer"
	"github.com/viphxin/xingo/udpserv"
	"github.com/viphxin/xingo/utils"

	//"syscall"
	"time"
)

var MAX_UINT32 uint32 = 4294967295
var PER_CNT uint32 = 5

func init() {
	utils.GlobalObject.Protoc = fnet.NewProtocol()
	// --------------------------------------------init log start
	//utils.ReSettingLog()
	// --------------------------------------------init log end
}

type Server struct {
	Port          int
	MaxConn       int
	sessIdPool    chan uint32
	connectionMgr iface.Iconnectionmgr
	timeChan      *timer.Timer
}

func NewServer() iface.Iserver {
	s := &Server{
		Port:    utils.GlobalObject.TcpPort,
		MaxConn: utils.GlobalObject.MaxConn,
	}
	s.connectionMgr = fnet.NewConnectionMgr(utils.GlobalObject.IsNet())
	s.initSessIdPool()
	utils.GlobalObject.TcpServer = s

	return s
}

func (this *Server) initSessIdPool() {
	go func() {
		this.sessIdPool = make(chan uint32, PER_CNT)
		ct := uint32(1)
		for {
			this.sessIdPool <- ct
			ct++
			if ct == MAX_UINT32 {
				ct = uint32(1)
			}
		}
	}()
}

func (this *Server) handleConnection(conn *net.TCPConn) {
	genNum := <-this.sessIdPool
	//logger.Prof("handleConnection genNum:", genNum)
	conn.SetNoDelay(true)
	conn.SetKeepAlive(true)
	conn.SetWriteBuffer(1024 * 1024)
	conn.SetReadBuffer(1024 * 1024)
	var key []byte = nil
	if utils.GlobalObject.IsNet() && utils.GlobalObject.Encrypt {
		key = this.connectionMgr.FetchEncKey()
	}
	fconn := fnet.NewConnection(conn, genNum, utils.GlobalObject.Protoc, key)
	if fconn.SyncKey() {
		fconn.Start()
	} else {
		logger.Errorf("conn encrypt check err id: %v", genNum)
		conn.Close()
	}
}

func (this *Server) Start() {
	utils.StartFastSec()
	go func() {
		utils.GlobalObject.Protoc.InitWorker(utils.GlobalObject.PoolSize)
		if utils.GlobalObject.IsGate() {
			utils.GlobalObject.ProtocGate.InitWorker(utils.GlobalObject.PoolSize)
		}
		if utils.GlobalObject.IsGame() {
			utils.GlobalObject.ProtocGate.InitWorker(utils.GlobalObject.PoolSize)
		}
		ln, err := net.ListenTCP("tcp", &net.TCPAddr{
			Port: this.Port,
		})
		if err != nil {
			//logger.Error(err)
			panic(err)
		}
		logger.Info("start xingo server...")
		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				logger.Error(err)
			}
			logger.Info("get conn: %v", conn)
			//max client exceed
			if this.connectionMgr.Len() >= utils.GlobalObject.MaxConn {
				logger.Error("fatal error, connection exceed")
				conn.Close()
			} else {
				go this.handleConnection(conn)
			}
		}
	}()
}

func (this *Server) GetConnectionMgr() iface.Iconnectionmgr {
	return this.connectionMgr
}

func (this *Server) GetConnectionQueue() chan interface{} {
	return nil
}

func (this *Server) Stop() {
	logger.Info("stop xingo server!!!")

	if utils.GlobalObject.IsGame() {
		udpserv.GlobalUdpServ.Close()
	}
	if utils.GlobalObject.OnServerStop != nil {
		utils.GlobalObject.OnServerStop()
	}
}

func (this *Server) AddRouter(router iface.IRouter) {
	utils.GlobalObject.Protoc.GetMsgHandle().AddRouter(router)
}

func (this *Server) CallWhen(ts string, f func(v ...interface{}), args ...interface{}) {
	loc, err_loc := time.LoadLocation("Local")
	if err_loc != nil {
		logger.Error(err_loc)
		return
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", ts, loc)
	now := time.Now()
	if err == nil {
		if now.Before(t) {
			this.CallLater(t.Sub(now), f, args...)
		} else {
			logger.Error("CallWhen time before now")
		}
	} else {
		logger.Error(err)
	}
}

func (this *Server) CallLater(durations time.Duration, f func(v ...interface{}), args ...interface{}) func() {
	if durations <= 0 {
		delayTask := timer.NewTimer(durations, f, args)
		utils.GlobalObject.TimeChan <- delayTask
		return func() {}
	}

	tr := time.AfterFunc(durations, func() {
		delayTask := timer.NewTimer(durations, f, args)
		utils.GlobalObject.TimeChan <- delayTask
	})

	return func() {
		tr.Stop()
	}
}

func (this *Server) CallLoop(durations time.Duration, f func(v ...interface{}), args ...interface{}) func() {
	ch := make(chan int, 1)
	tr := time.NewTicker(durations)
	go func() {
		delayTask := timer.NewTimer(durations, f, args)
		for {
			select {
			case <-ch:
				tr.Stop()
				close(ch)
				return
			case <-tr.C:
				utils.GlobalObject.TimeChan <- delayTask
			}
		}
	}()

	return func() {
		ch <- 0
	}
}

func (this *Server) WaitSignal() {
	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	//signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGUSR1)
	sig := <-c
	logger.Info(fmt.Sprintf("server exit. signal: [%s]", sig))
	this.Stop()
}

func (this *Server) Serve() {
	this.Start()
	this.WaitSignal()
}
