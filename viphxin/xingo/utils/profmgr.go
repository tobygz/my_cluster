package utils

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

var GlobalProfMgr *profMgr

func init() {
	GlobalProfMgr = NewProfMgr(time.Second)
}

type profElem struct {
	id      int
	ct      uint64
	startMs time.Time
	totalMs time.Duration
	sync.RWMutex
}

func (this *profElem) RStart() {
	this.Lock()
	defer this.Unlock()
	this.startMs = time.Now()
}

func (this *profElem) REnd() {
	this.Lock()
	defer this.Unlock()
	this.ct = this.ct + 1
	diff := time.Since(this.startMs)
	this.totalMs = this.totalMs + diff
	//fmt.Printf("profElem id: %d ct: %d, totalMs: %d, add: %d goid: %d\r\n", this.id, this.ct, this.totalMs, diff, Goid())
}

func (this *profElem) Reset() {
	this.Lock()
	defer this.Unlock()
	this.ct = 0
	//this.startMs = 0
	this.totalMs = 0
	//fmt.Println()
}

func (this *profElem) ToString() string {
	this.Lock()
	defer this.Unlock()
	if this.ct == 0 {
		return ""
	}
	ret := fmt.Sprintf(" prof elem id: %d ct: %d avg: %f use time: %d ", this.id, this.ct, float64(this.totalMs)/float64(int64(this.ct)), this.totalMs)
	return ret
}

type profMgr struct {
	objMap   map[int]*profElem
	interval time.Duration
	lastMs   time.Time
	sync.RWMutex
}

func NewProfMgr(intv time.Duration) *profMgr {
	return &profMgr{
		objMap:   make(map[int]*profElem),
		interval: intv,
		lastMs:   time.Now(),
	}
}

func (this *profMgr) MakeAdd(val int) *profElem {
	this.Lock()
	defer this.Unlock()
	if objg, ok := this.objMap[val]; ok {
		return objg
	}
	obj := &profElem{
		id: val,
		ct: uint64(0),
	}
	this.objMap[val] = obj
	return obj
}

func (this *profMgr) Dump() (bool, string) {
	this.Lock()
	defer this.Unlock()
	nowMs := time.Now()
	diff := nowMs.Sub(this.lastMs)
	if diff < this.interval {
		return false, ""
	}

	this.lastMs = nowMs

	var keys []int
	for k, _ := range this.objMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	retStr := "==============================================================================================================\r\n"
	for _, k := range keys {
		elem := this.objMap[k]
		retStr = fmt.Sprintf("%s\r\n%s", retStr, elem.ToString())
		elem.Reset()
	}
	retStr = fmt.Sprintf("%s\r\n==============================================================================================================\r\n", retStr)
	return true, retStr
}
