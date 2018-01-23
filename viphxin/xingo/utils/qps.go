package utils

import (
	"fmt"
	"time"
)

type QpsMgr struct {
	startMs   time.Time
	totalCt   int64
	totalSize int64
	interval  time.Duration
}

func NewQps(msec time.Duration) *QpsMgr {
	if msec == 0 {
		msec = 10 * time.Millisecond
	}
	return &QpsMgr{
		startMs:   time.Now(),
		totalCt:   int64(0),
		totalSize: int64(0),
		interval:  msec,
	}
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
	ret := fmt.Sprintf("QpsMgr dump total: %d qps: %v totalSize: %v timeUse: %v(sec)", this.totalCt, float64(this.totalCt)/float64(timeUse), this.totalSize, timeUse)
	this.totalCt = int64(0)
	this.totalSize = int64(0)
	this.startMs = nowMs
	return true, ret, retTotal
}
