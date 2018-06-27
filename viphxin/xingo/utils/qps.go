package utils

import (
	"fmt"
	"sync/atomic"
	"time"
)

type QpsMgr struct {
	startMs    time.Time
	totalCt    int64
	totalSize  int64
	NetOutByte int64
	NetInByte  int64
	interval   time.Duration
}

func NewQps(msec time.Duration) *QpsMgr {
	if msec == 0 {
		msec = 10 * time.Millisecond
	}
	return &QpsMgr{
		startMs:    time.Now(),
		totalCt:    int64(0),
		totalSize:  int64(0),
		NetOutByte: int64(0),
		NetInByte:  int64(0),
		interval:   msec,
	}
}

func (this *QpsMgr) UpdateNetIn(size int) {
	atomic.AddInt64(&this.NetInByte, int64(size))
}

func (this *QpsMgr) FetchNetIn() int64 {
	val := atomic.LoadInt64(&this.NetInByte)
	atomic.StoreInt64(&this.NetInByte, 0)
	return val
}

func (this *QpsMgr) UpdateNetOut(size int) {
	atomic.AddInt64(&this.NetOutByte, int64(size))
}

func (this *QpsMgr) FetchNetOut() int64 {
	val := atomic.LoadInt64(&this.NetOutByte)
	atomic.StoreInt64(&this.NetOutByte, 0)
	return val
}

func (this *QpsMgr) Add(ct int, size int) {
	this.totalCt = int64(ct) + this.totalCt
	this.totalSize = int64(size) + this.totalSize
}

func (this *QpsMgr) Dump() (bool, string, int64) {
	nowMs := time.Now()
	diff := nowMs.Sub(this.startMs)
	if diff < this.interval {
		return false, "", int64(0)
	}

	retTotal := this.totalCt
	timeUse := float64(diff) / float64(time.Second)
	ret := fmt.Sprintf("QpsMgr dump total: %d qps: %v totalSize: %v  netin: %d netout: %d timeUse: %v(sec)",
		this.totalCt, float64(this.totalCt)/float64(timeUse), this.totalSize, this.FetchNetIn(), this.FetchNetOut(), timeUse)
	this.totalCt = int64(0)
	this.totalSize = int64(0)
	this.startMs = nowMs
	return true, ret, retTotal
}
