package cluster

/*
	regest rpc
*/
import (
	"fmt"
	"github.com/viphxin/xingo/logger"
	"github.com/viphxin/xingo/utils"
	"math/rand"
	"reflect"
	"time"
)

type RpcMsgHandle struct {
	PoolSize  int32
	TaskQueue []chan *RpcRequest
	Apis      map[string]reflect.Value
}

func NewRpcMsgHandle() *RpcMsgHandle {
	return &RpcMsgHandle{
		PoolSize:  utils.GlobalObject.PoolSize,
		TaskQueue: make([]chan *RpcRequest, utils.GlobalObject.PoolSize),
		Apis:      make(map[string]reflect.Value),
	}
}

/*
处理rpc消息
*/
func (this *RpcMsgHandle) DoMsg(request *RpcRequest) {

	//logger.Debug(fmt.Sprintf("+++++++++ rpc: %s now_name: %s ", request.Rpcdata.Target, utils.GlobalObject.Name))
	if request.Rpcdata.MsgType == RESPONSE && request.Rpcdata.Key != "" {
		//放回异步结果
		AResultGlobalObj.FillAsyncResult(request.Rpcdata.Key, request.Rpcdata)
		return
	} else {
		//rpc 请求
		if f, ok := this.Apis[request.Rpcdata.Target]; ok {
			//存在
			st := time.Now()
			if request.Rpcdata.MsgType == REQUEST_FORRESULT {
				ret := f.Call([]reflect.Value{reflect.ValueOf(request)})
				rpcdata := &RpcData{
					MsgType: RESPONSE,
					Result:  ret[0].Interface().(string),
					Key:     request.Rpcdata.Key,
				}
				request.Fconn.Send(rpcdata.Encode())
			} else if request.Rpcdata.MsgType == REQUEST_NORESULT {
				f.Call([]reflect.Value{reflect.ValueOf(request)})
			}

			if utils.GlobalObject.EnableFlowLog {
				logger.Debug(fmt.Sprintf("rpc %s cost total time: %f ms", request.Rpcdata.Target, time.Now().Sub(st).Seconds()*1000))
			}
		} else {
			logger.Error(fmt.Sprintf("not found rpc:  %s", request.Rpcdata.Target))
		}
	}
}

func (this *RpcMsgHandle) DeliverToMsgQueue(pkg interface{}) {
	request := pkg.(*RpcRequest)
	//add to worker pool
	index := rand.Int31n(utils.GlobalObject.PoolSize)
	taskQueue := this.TaskQueue[index]
	if utils.GlobalObject.EnableFlowLog {
		logger.Debug(fmt.Sprintf("add to rpc pool : %d ", index))
	}
	taskQueue <- request
	logger.Trace(fmt.Sprintf("DeliverToMsgQueue add to rpc pool : %d size: %d", index, len(taskQueue)))
}

func (this *RpcMsgHandle) DoMsgFromGoRoutine(pkg interface{}) {
	request := pkg.(*RpcRequest)
	go this.DoMsg(request)
}

func (this *RpcMsgHandle) AddRouter(router interface{}) {
	value := reflect.ValueOf(router)
	tp := value.Type()
	for i := 0; i < value.NumMethod(); i += 1 {
		name := tp.Method(i).Name

		if _, ok := this.Apis[name]; ok {
			//存在
			panic("repeated rpc " + name)
		}
		this.Apis[name] = value.Method(i)
		logger.Info(fmt.Sprintf("now_name: %s add RpcMsgHandle: %s ", utils.GlobalObject.Name, name))
	}
}

func (this *RpcMsgHandle) StartWorkerLoop(poolSize int) {
	for i := 0; i < poolSize; i += 1 {
		c := make(chan *RpcRequest, utils.GlobalObject.MaxWorkerLen)
		this.TaskQueue[i] = c
		go func(index int, taskQueue chan *RpcRequest) {
			logger.Info(fmt.Sprintf("init rpc thread pool %d.", index))
			for {
				request := <-taskQueue
				this.DoMsg(request)
			}
			logger.Info(fmt.Sprintf("rpc thread %d. exit", index))

		}(i, c)
	}
}
