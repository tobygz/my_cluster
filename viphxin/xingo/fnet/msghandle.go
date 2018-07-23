package fnet

//#include <unistd.h>
// typedef void (*yieldFunc) ();
//
// void
// bridge_yield_func(yieldFunc f)
// {
//      return f();
// }
//
// void cyield()
// {
//      usleep(11*1000);
//      return ;
// }
// void cyieldMs(int val)
// {
//      usleep(val);
//      return ;
// }
import "C"

/*
	find msg api
*/
import (
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var GAME_WORK_GOR_NUM int = 4

type MsgHandle struct {
	PoolSize  int32
	TaskQueue []chan *PkgAll
	Apis      map[uint32]func(iface.IRequest, uint32, uint32)
	GApis     map[uint32]func(iface.IRequest, uint32, uint64)
	QpsObj    *utils.QpsMgr
	sync.RWMutex
}

func NewMsgHandle() *MsgHandle {
	return &MsgHandle{
		PoolSize:  utils.GlobalObject.PoolSize,
		TaskQueue: make([]chan *PkgAll, utils.GlobalObject.PoolSize),
		Apis:      make(map[uint32]func(iface.IRequest, uint32, uint32)),
		GApis:     make(map[uint32]func(iface.IRequest, uint32, uint64)),
		QpsObj:    utils.NewQps(time.Second),
	}
}

func (this *MsgHandle) UpdateNetIn(size int) {
	this.QpsObj.UpdateNetIn(size)
}

func (this *MsgHandle) UpdateNetOut(size int) {
	this.QpsObj.UpdateNetOut(size)
}

func (this *MsgHandle) Name() string {
	return "MsgHandle"
}

//一致性路由,保证同一连接的数据转发给相同的goroutine
func (this *MsgHandle) DeliverToMsgQueue(pkg interface{}) {
	data := pkg.(*PkgAll)

	if utils.GlobalObject.IsGate() {
		this.Raw_DeliverToMsgQueue(data, 0)
	} else if utils.GlobalObject.IsGame() {
		idx := uint32(data.Pid) % uint32(GAME_WORK_GOR_NUM)
		this.Raw_DeliverToMsgQueue(data, idx)
	} else if utils.GlobalObject.IsNet() {
		index := uint32(data.Fconn.GetSessionId()) % uint32(utils.GlobalObject.PoolSize)
		this.Raw_DeliverToMsgQueue(data, index)
	}
}

func (this *MsgHandle) Raw_DeliverToMsgQueue(data *PkgAll, index uint32) {
	//this.Lock()
	//defer this.Unlock()
	taskQueue := this.TaskQueue[index]
	//if utils.GlobalObject.EnableFlowLog {
	//logger.Debug(fmt.Sprintf("add to pool : %d", index))
	//}
	taskQueue <- data
}

func (this *MsgHandle) DoMsgFromGoRoutine(pkg interface{}) {
	data := pkg.(*PkgAll)
	go func() {
		if f, ok := this.Apis[data.Pdata.MsgId]; ok {
			//存在
			f(data, data.Pdata.MsgId, uint32(data.Pid))

		} else {
			logger.Error(fmt.Sprintf("not found api:  %d", data.Pdata.MsgId))
		}
	}()
}

func (this *MsgHandle) AddRouter(router iface.IRouter) {
	/*
		for i := 0; i < 1000; i++ {
			//to net
		}
	*/
	apiMap := router.GetApiMap()
	if strings.Contains(utils.GlobalObject.Name, "net") {
		var func_msg2gate func(iface.IRequest, uint32, uint32)
		var func_msg2game func(iface.IRequest, uint32, uint32)
		for name, method := range apiMap {
			if strings.Contains(name, "Api_msg2gate") == true {
				func_msg2gate = method
			}
			if strings.Contains(name, "Api_msg2game") == true {
				func_msg2game = method
			}
		}

		for i := 0; i < 1999; i++ {
			//to gate
			this.Apis[uint32(i)] = func_msg2gate
		}
		for i := 2000; i < 2999; i++ {
			this.Apis[uint32(i)] = func_msg2game
			//to gate
			//this.Apis[uint32(i)] = router.Api_msg2gate
		}
		for i := 10001; i < 12000; i++ {
			//to global
			//this.Apis[uint32(i)] = router.Api_msg2global
		}
	}
	this.AddRouter_raw(router)
}

func (this *MsgHandle) AddRpcRouter(router iface.IRpcRouter) {
}

func (this *MsgHandle) AddRouter_raw(router iface.IRouter) {
	if utils.GlobalObject.IsGame() {
		apiMap := router.GetGApiMap()
		for name, method := range apiMap {
			if strings.Contains(name, "Api") != true {
				logger.Info(fmt.Sprintf("router ignore contain func: %s", name))
				continue
			}
			k := strings.Split(name, "_")
			index, err := strconv.Atoi(k[len(k)-1])
			if err != nil {
				//panic("error api: " + name)
				logger.Debug("ignore func", name)
				continue
			}
			if _, ok := this.GApis[uint32(index)]; ok {
				//存在
				panic("repeated api " + string(index))
			}
			this.GApis[uint32(index)] = method
			logger.Info(fmt.Sprintf("msghandle add api idx: %d name:%s", index, name))
		}
	} else {
		apiMap := router.GetApiMap()
		for name, method := range apiMap {
			if strings.Contains(name, "Api") != true {
				logger.Info(fmt.Sprintf("router ignore contain func: %s", name))
				continue
			}
			k := strings.Split(name, "_")
			index, err := strconv.Atoi(k[len(k)-1])
			if err != nil {
				//panic("error api: " + name)
				logger.Debug("ignore func", name)
				continue
			}
			if _, ok := this.Apis[uint32(index)]; ok {
				//存在
				panic("repeated api " + string(index))
			}
			this.Apis[uint32(index)] = method
			logger.Info(fmt.Sprintf("msghandle add api idx: %d name:%s", index, name))
		}
	}

}

func (this *MsgHandle) NetWorkerLoop(i int, c chan *PkgAll) {
	go func(index int, taskQueue chan *PkgAll) {
		logger.Info(fmt.Sprintf("NetWorkerLoop init thread pool %d.", index))
		var msgId uint32
		var pid uint32
		for {
			select {
			case data := <-taskQueue:
				if f, ok := this.Apis[data.Pdata.MsgId]; ok {
					//logger.Debug(fmt.Sprintf("Api_%d called ", data.Pdata.MsgId))
					msgId = data.Pdata.MsgId
					pid = uint32(data.Pid)
					utils.XingoTry(f, this.HandleError, data, msgId, pid)
					this.QpsObj.Add(1, 1)
					flag, info, _ := this.QpsObj.Dump()
					if flag {
						logger.Prof(fmt.Sprintf("NetWorkerLoop idx: %d %s runtime status: %s", index, info, utils.GetRuntimeStatus()))
					}
				} else {
					logger.Error(fmt.Sprintf("not found api:  %d", data.Pdata.MsgId))
				}
			case df := <-utils.GlobalObject.TimeChan:
				df.GetFunc().Call()
			}
		}
	}(i, c)

}

func (this *MsgHandle) GameWorkerLoop(i int, c chan *PkgAll) {
	/*
		chsleep := make(chan int, 1)
		go func() {
			sleepf := C.yieldFunc(C.cyield)
			for {
				C.bridge_yield_func(sleepf)
				chsleep <- int(0)
			}
		}()
	*/

	go func(index int, taskQueue chan *PkgAll) {
		logger.Info(fmt.Sprintf("GameWorkerLoop init thread pool %d.", index))
		var msgId uint32
		var pid uint64
		runtime.LockOSThread()
		for {
			if utils.GlobalObject.WebObj == nil {
				continue
			}
			select {
			case data := <-taskQueue:
				if f, ok := this.GApis[data.Pdata.MsgId]; ok {
					logger.Debug(fmt.Sprintf("Api_%d called ", data.Pdata.MsgId))
					msgId = data.Pdata.MsgId
					pid = data.Pid
					utils.XingoTry64(f, this.HandleError, data, msgId, pid)
					this.QpsObj.Add(1, 1)
					flag, info, _ := this.QpsObj.Dump()
					if flag {
						logger.Prof(fmt.Sprintf("GameWorkerLoop idx: %d %s runtime status: %s", index, info, utils.GetRuntimeStatus()))
					}
				} else {
					logger.Error(fmt.Sprintf("not found api:  %d", data.Pdata.MsgId))
				}
			case req := <-utils.GlobalObject.WebObj.GetReqChan():
				if utils.GlobalObject.WebObj != nil && req != nil {
					utils.GlobalObject.WebObj.HandleReqCall(req)
				}
			case df := <-utils.GlobalObject.TimeChan:
				df.GetFunc().Call()
				//case <-tick.C:
			default:
				syscall.Nanosleep(&syscall.Timespec{int64(0), int64(1000000 * 33)}, nil)
				if utils.GlobalObject.OnServerMsTimer != nil {
					utils.GlobalObject.OnServerMsTimer()
				}
			}
		}
	}(i, c)
}

func (this *MsgHandle) GateWorkerLoop(i int, c chan *PkgAll) {
	go func(index int, taskQueue chan *PkgAll) {
		logger.Info(fmt.Sprintf("GateWorkerLoop init thread pool %d.", index))
		var msgId uint32
		var pid uint32
		for {
			if utils.GlobalObject.WebObj == nil {
				continue
			}
			select {
			case data := <-taskQueue:
				if f, ok := this.Apis[data.Pdata.MsgId]; ok {
					//存在
					if data.Pdata.MsgId != 1001 {
						logger.Debug(fmt.Sprintf("Api_%d called ", data.Pdata.MsgId))
					}
					msgId = data.Pdata.MsgId
					pid = uint32(data.Pid)
					utils.XingoTry(f, this.HandleError, data, msgId, pid)
					this.QpsObj.Add(1, 1)
					flag, info, _ := this.QpsObj.Dump()
					if flag {
						logger.Prof(fmt.Sprintf("GateWorkerLoop idx: %d %s runtime status: %s", index, info, utils.GetRuntimeStatus()))
					}
				} else {
					logger.Error(fmt.Sprintf("not found api:  %d", data.Pdata.MsgId))
				}

			case req := <-utils.GlobalObject.WebObj.GetReqChan():
				if utils.GlobalObject.WebObj != nil && req != nil {
					utils.GlobalObject.WebObj.HandleReqCall(req)
				}
			case df := <-utils.GlobalObject.TimeChan:
				df.GetFunc().Call()
			}
		}
	}(i, c)

}

func (this *MsgHandle) StartWorkerLoop(poolSize int) {
	logger.Debug("called StartWorkerLoop inner")
	if utils.GlobalObject.IsGate() {
		c := make(chan *PkgAll, utils.GlobalObject.MaxWorkerLen)
		this.TaskQueue[0] = c
		this.GateWorkerLoop(0, c)
	} else if utils.GlobalObject.IsGame() {
		this.TaskQueue = make([]chan *PkgAll, GAME_WORK_GOR_NUM)
		for i := 0; i < GAME_WORK_GOR_NUM; i++ {
			c := make(chan *PkgAll, utils.GlobalObject.MaxWorkerLen)
			this.TaskQueue[i] = c
			this.GameWorkerLoop(i, c)
		}
	} else {
		for i := 0; i < int(utils.GlobalObject.PoolSize); i += 1 {
			c := make(chan *PkgAll, 1)
			this.TaskQueue[i] = c
			this.NetWorkerLoop(i, c)
		}
	}
}

func (this *MsgHandle) HandleError(err interface{}) {
	if err != nil {
		logger.Error(err)
		buf := make([]byte, 1<<16)
		stackSize := runtime.Stack(buf, true)
		logger.Error(fmt.Sprintf("%s\n", string(buf[0:stackSize])))
	}
}
