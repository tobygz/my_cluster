package utils

import (
	"sync/atomic"
	"time"
)

var g_tick uint32 = uint32(time.Now().Unix())

func StartFastSec() {
	go func() {
		c1 := make(chan uint32, 1)
		tick := time.NewTicker(time.Second)
		for {
			select {
			case val := <-c1:
				atomic.StoreUint32(&g_tick, val)
			case <-tick.C:
				valt := uint32(time.Now().Unix())
				c1 <- valt
			}
		}
	}()
}

func GetFastSec() uint32 {
	nowVal := atomic.LoadUint32(&g_tick)
	return nowVal
}
