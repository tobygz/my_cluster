package timer

import (
	//"github.com/viphxin/xingo/logger"
	//"math"
	//"fmt"
	"sync"
	"time"
)

/*
协程安全的timer
*/
const (
	//默认安全时间调度器的容量
	TIMERLEN = 2048
	//默认最大误差值10毫秒
	ERRORMAX = 1
	//默认最大触发队列缓冲大小
	TRIGGERMAX = 1024
	//默认hashwheel分级
	LEVEL = 36
)

func UnixTS() int64 {
	return time.Now().UnixNano() / 1e6
}

type ParamNull struct{}

type SafeTimer struct {
	//延迟调用的函数
	delayCall *DelayCall
	//调用的时间：单位毫秒
	unixts int64
	//
	interval int64
	tid      uint32
}

func (this *SafeTimer) ResetUnixts() {
	//fmt.Println("ResetUnixts interval: %d", this.interval)
	//nowTs := UnixTS()
	//logger.Debug("want call ResetUnixts:", this.unixts, ", real nowts:", nowTs, ",diff:", (nowTs - this.unixts))
	this.unixts = UnixTS() + this.interval
}

func (this *SafeTimer) SetTid(v uint32) {
	this.tid = v
}

func (this *SafeTimer) GetTid() uint32 {
	return this.tid
}

func (this *SafeTimer) GetTs() int64 {
	return this.unixts
}

func NewSafeTimer(delay int64, loop bool, delayCall *DelayCall) *SafeTimer {

	unixts := UnixTS()
	//fmt.Println("NewSafeTimer interval: ", unixts)
	if delay > 0 {
		unixts += delay
	}
	if !loop {
		delay = 0
	}
	return &SafeTimer{
		delayCall: delayCall,
		unixts:    unixts,
		interval:  delay,
	}
}

type SafeTimerScheduel struct {
	hashwheel   *HashWheel
	idGen       uint32
	triggerChan chan *DelayCall
	sync.RWMutex
}

func NewSafeTimerScheduel() *SafeTimerScheduel {
	scheduel := &SafeTimerScheduel{
		hashwheel:   NewHashWheel("wheel_hours", LEVEL, 3600*1e3, TIMERLEN),
		idGen:       0,
		triggerChan: make(chan *DelayCall, TRIGGERMAX),
	}

	//minute wheel
	minuteWheel := NewHashWheel("wheel_minutes", LEVEL, 60*1e3, TIMERLEN)
	//second wheel
	secondWheel := NewHashWheel("wheel_seconds", LEVEL, 1*1e3, TIMERLEN)
	minuteWheel.AddNext(secondWheel)
	scheduel.hashwheel.AddNext(minuteWheel)

	go scheduel.StartScheduelLoop()
	return scheduel
}

func (this *SafeTimerScheduel) GetTriggerChannel() chan *DelayCall {
	return this.triggerChan
}

func (this *SafeTimerScheduel) CreateTimer(delay int64, f func(v ...interface{}), args []interface{}, loop bool) (uint32, error) {
	this.Lock()
	defer this.Unlock()

	this.idGen += 1
	err := this.hashwheel.Add2WheelChain(this.idGen,
		NewSafeTimer(delay, loop, &DelayCall{
			f:    f,
			args: args,
		}))
	if err != nil {
		return 0, err
	} else {
		return this.idGen, nil
	}
}

func (this *SafeTimerScheduel) CancelTimer(timerId uint32) {
	this.hashwheel.RemoveFromWheelChain(timerId)
}

func (this *SafeTimerScheduel) StartScheduelLoop() {
	//logger.Info("xingo safe timer scheduelloop runing.")
	for {
		triggerList := this.hashwheel.GetTriggerWithIn(ERRORMAX)
		//trigger
		for _, v := range triggerList {
			//logger.Debug("want call: ", v.unixts, ".real call: ", UnixTS(), ".ErrorMS: ", UnixTS()-v.unixts)
			/*
				if math.Abs(float64(UnixTS()-v.unixts)) > float64(10){
					logger.Error("want call: ", v.unixts, ".real call: ", UnixTS(), ".ErrorMS: ", UnixTS()-v.unixts)
				}
			*/
			this.triggerChan <- v.delayCall
		}

		//wait for next loop
		time.Sleep(ERRORMAX * time.Millisecond)
	}
}
