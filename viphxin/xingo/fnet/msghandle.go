package fnet

/*
	find msg api
*/
import (
	"fmt"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/udpserv"
	"github.com/viphxin/xingo/utils"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type MsgHandle struct {
	PoolSize  int32
	TaskQueue []chan *PkgAll
	Apis      map[uint32]reflect.Value
}

func NewMsgHandle() *MsgHandle {
	return &MsgHandle{
		PoolSize:  utils.GlobalObject.PoolSize,
		TaskQueue: make([]chan *PkgAll, utils.GlobalObject.PoolSize),
		Apis:      make(map[uint32]reflect.Value),
	}
}

//一致性路由,保证同一连接的数据转发给相同的goroutine
func (this *MsgHandle) DeliverToMsgQueue(pkg interface{}) {
	data := pkg.(*PkgAll)

	//index := uint32(data.Pid) % uint32(utils.GlobalObject.PoolSize)
	if utils.GlobalObject.IsGate() {
		this.Raw_DeliverToMsgQueue(data, 0)
	} else if utils.GlobalObject.IsNet() {
		index := uint32(data.Fconn.GetSessionId()) % uint32(utils.GlobalObject.PoolSize)
		this.Raw_DeliverToMsgQueue(data, index)
	}
}

func (this *MsgHandle) Raw_DeliverToMsgQueue(data *PkgAll, index uint32) {
	//index := rand.Int31n(utils.GlobalObject.PoolSize)
	taskQueue := this.TaskQueue[index]
	if utils.GlobalObject.EnableFlowLog {
		logger.Debug(fmt.Sprintf("add to pool : %d", index))
	}
	taskQueue <- data
}

func (this *MsgHandle) DoMsgFromGoRoutine(pkg interface{}) {
	data := pkg.(*PkgAll)
	go func() {
		if f, ok := this.Apis[data.Pdata.MsgId]; ok {
			//存在
			st := time.Now()
			f.Call([]reflect.Value{reflect.ValueOf(data)})

			if utils.GlobalObject.EnableFlowLog {
				logger.Debug(fmt.Sprintf("Api_%d cost total time: %f ms", data.Pdata.MsgId, time.Now().Sub(st).Seconds()*1000))
			}
		} else {
			logger.Error(fmt.Sprintf("not found api:  %d", data.Pdata.MsgId))
		}
	}()
}

func (this *MsgHandle) AddRouter(router interface{}) {
	/*
		for i := 0; i < 1000; i++ {
			//to net
		}
	*/
	if strings.Contains(utils.GlobalObject.Name, "net") {
		value := reflect.ValueOf(router)
		tp := value.Type()
		var func_msg2gate reflect.Value
		for i := 0; i < value.NumMethod(); i += 1 {
			name := tp.Method(i).Name
			if strings.Contains(name, "Api_msg2gate") == true {
				func_msg2gate = value.Method(i)
			}
		}

		for i := 0; i < 3000; i++ {
			//to gate
			this.Apis[uint32(i)] = func_msg2gate
		}
		for i := 3000; i < 5000; i++ {
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

func (this *MsgHandle) AddRouter_raw(router interface{}) {
	value := reflect.ValueOf(router)
	tp := value.Type()
	for i := 0; i < value.NumMethod(); i += 1 {
		name := tp.Method(i).Name
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
		this.Apis[uint32(index)] = value.Method(i)
		logger.Info(fmt.Sprintf("msghandle add api idx: %d name:%s", index, name))
	}

	//exec test
	// for i := 0; i < 100; i += 1 {
	// 	Apis[1].Call([]reflect.Value{reflect.ValueOf("huangxin"), reflect.ValueOf(router)})
	// 	Apis[2].Call([]reflect.Value{})
	// }
	// fmt.Println(this.Apis)
	// this.Apis[2].Call([]reflect.Value{reflect.ValueOf(&PkgAll{})})
}

func (this *MsgHandle) NetWorkerLoop(i int, c chan *PkgAll) {
	go func(index int, taskQueue chan *PkgAll) {
		logger.Info(fmt.Sprintf("NetWorkerLoop init thread pool %d.", index))
		var msgId uint32
		var pid uint32

		var st uint32

		lastSec := utils.GetFastSec()
		lenVal := 0

		for {
			st = utils.GetFastSec()
			if st-lastSec >= uint32(10) {
				logger.Profile(fmt.Sprintf("NetWorkerLoop index: %d taskQueue leftLen: %d lenVal: %d qps: %d", index, len(taskQueue), lenVal, uint32(lenVal/10)))
				lastSec = st
				lenVal = 0
			}
			lenVal++

			select {
			case data := <-taskQueue:
				if f, ok := this.Apis[data.Pdata.MsgId]; ok {
					//存在
					st := time.Now()

					//logger.Debug(fmt.Sprintf("Api_%d called ", data.Pdata.MsgId))
					//f.Call([]reflect.Value{reflect.ValueOf(data)})
					msgId = data.Pdata.MsgId
					pid = data.Pid
					utils.XingoTry(f, []reflect.Value{reflect.ValueOf(data), reflect.ValueOf(msgId), reflect.ValueOf(pid)}, this.HandleError)
					if utils.GlobalObject.EnableFlowLog {
						logger.Debug(fmt.Sprintf("Api_%d cost total time: %f ms", data.Pdata.MsgId, time.Now().Sub(st).Seconds()*1000))
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

func (this *MsgHandle) GateWorkerLoop(i int, c chan *PkgAll) {
	go func(index int, taskQueue chan *PkgAll) {
		logger.Info(fmt.Sprintf("GateWorkerLoop init thread pool %d.", index))
		var msgId uint32
		var pid uint32
		for {
			if udpserv.GlobalUdpServ == nil {
				udpserv.NewUdpServ(utils.GlobalObject.UdpPort)
			}
			if utils.GlobalObject.WebObj == nil {
				continue
			}
			select {
			case data := <-taskQueue:
				if f, ok := this.Apis[data.Pdata.MsgId]; ok {
					//存在
					st := time.Now()

					logger.Debug(fmt.Sprintf("Api_%d called ", data.Pdata.MsgId))
					//f.Call([]reflect.Value{reflect.ValueOf(data)})
					msgId = data.Pdata.MsgId
					pid = data.Pid
					utils.XingoTry(f, []reflect.Value{reflect.ValueOf(data), reflect.ValueOf(msgId), reflect.ValueOf(pid)}, this.HandleError)
					if utils.GlobalObject.EnableFlowLog {
						logger.Debug(fmt.Sprintf("Api_%d cost total time: %f ms", data.Pdata.MsgId, time.Now().Sub(st).Seconds()*1000))
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
			case udpreq := <-udpserv.GlobalUdpServ.GetChan():
				msgId := uint32(112)
				f, ok := this.Apis[msgId]
				if !ok {
					panic("fetch 112 handler fail")
				}
				if udpserv.GlobalUdpServ.Running {
					pid = 0
					utils.XingoTry(f, []reflect.Value{reflect.ValueOf(udpreq), reflect.ValueOf(msgId), reflect.ValueOf(pid)}, this.HandleError)
				}

			}
		}
	}(i, c)

}

func (this *MsgHandle) StartWorkerLoop(poolSize int) {

	if utils.GlobalObject.IsGate() {
		c := make(chan *PkgAll, utils.GlobalObject.MaxWorkerLen)
		this.TaskQueue[0] = c
		this.GateWorkerLoop(0, c)
	} else {
		for i := 0; i < int(utils.GlobalObject.PoolSize); i += 1 {
			c := make(chan *PkgAll, utils.GlobalObject.MaxWorkerLen)
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
