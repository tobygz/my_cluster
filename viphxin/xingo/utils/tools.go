package utils

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"github.com/viphxin/xingo/iface"
	"github.com/viphxin/xingo/logger"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func HttpRequestWrap(uri string, targat func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(response http.ResponseWriter, request *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				logger.Info("===================http server panic recover===============")
			}
		}()
		st := time.Now()
		logger.Debug("User-Agent: ", request.Header["User-Agent"])
		targat(response, request)
		logger.Debug(fmt.Sprintf("%s cost total time: %f ms", uri, time.Now().Sub(st).Seconds()*1000))
	}
}

func ReSettingLog() {
	// --------------------------------------------init log start
	logger.SetConsole(GlobalObject.SetToConsole)
	logger.SetFileMaxSize(GlobalObject.MaxFileSize, GlobalObject.LogFileUnit)
	logger.SetToSyslog(GlobalObject.ToSyslog)
	logger.SetSyslogAddr(GlobalObject.SyslogAddr, GlobalObject.SyslogPort)
	logger.SetLogFileLine(GlobalObject.LogFileLine)
	logger.SetLevel(GlobalObject.LogLevel)
	if GlobalObject.LogFileType == logger.ROLLINGFILE {
		logger.SetRollingFile(GlobalObject.LogPath, GlobalObject.LogName)
	} else {
		logger.SetRollingDaily(GlobalObject.LogPath, GlobalObject.LogName)
	}
	// --------------------------------------------init log end
}

func XingoTry64(f func(iface.IRequest, uint32, uint64), handler func(interface{}), data iface.IRequest, msgId uint32, pid uint64) {
	defer func() {
		if err := recover(); err != nil {
			logger.Info("-------------panic recover---------------")
			if handler != nil {
				handler(err)
			}

			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, true)

			logger.Error(fmt.Sprintf("%s\n", string(buf[0:stackSize])))
		}
	}()
	f(data, msgId, pid)
}

func XingoTry(f func(iface.IRequest, uint32, uint32), handler func(interface{}), data iface.IRequest, msgId, pid uint32) {
	defer func() {
		if err := recover(); err != nil {
			logger.Info("-------------panic recover---------------")
			if handler != nil {
				handler(err)
			}

			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, true)

			logger.Error(fmt.Sprintf("%s\n", string(buf[0:stackSize])))
		}
	}()
	f(data, msgId, pid)
}

func ZlibCompress(src []byte) []byte {
	var in bytes.Buffer
	w := zlib.NewWriter(&in)
	w.Write(src)
	w.Close()
	return in.Bytes()
}

func ZlibUnCompress(compressSrc []byte) []byte {
	b := bytes.NewReader(compressSrc)
	var out bytes.Buffer
	r, _ := zlib.NewReader(b)
	io.Copy(&out, r)
	return out.Bytes()
}

func Goid() int {
	return 0
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("panic recover:panic info:%v", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func GetRuntimeStatus() string {
	memst := &runtime.MemStats{}
	runtime.ReadMemStats(memst)
	ret := fmt.Sprintf("MemAlloc: %d Sys: %d Lookups: %d Mallocs: %d Frees: %d HeapAlloc: %d HeapSys: %d HeapIdle: %d HeapInuse: %d HeapReleased: %d HeapObjects: %d StackInuse: %d StackSys: %d MSpanInuse: %d MSpanSys: %d MCacheInuse: %d MCacheSys: %d BuckHashSys: %d GCSys: %d OtherSys: %d  numGoroutine: %d",
		memst.Alloc, memst.Sys, memst.Lookups, memst.Mallocs, memst.Frees, memst.HeapAlloc, memst.HeapSys, memst.HeapIdle, memst.HeapInuse, memst.HeapReleased, memst.HeapObjects, memst.StackInuse, memst.StackSys, memst.MSpanInuse, memst.MSpanSys, memst.MCacheInuse, memst.MCacheSys, memst.BuckHashSys, memst.GCSys, memst.OtherSys, runtime.NumGoroutine())
	return ret
}

/*
func GetRuntimeStatus() string {
	memst := &runtime.MemStats{}
	runtime.ReadMemStats(memst)
	ret := fmt.Sprintf("MemAlloc: %d numGoroutine: %d", memst.Alloc, runtime.NumGoroutine())
	return ret
}
*/

func PrintStack() {
	debug.PrintStack()
}

var g_rand *rand.Rand

func initRandSource() {
	g_rand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GetRandVal(limit int) int {
	if g_rand == nil {
		initRandSource()
	}
	return g_rand.Intn(limit)
}

func GetRandUVal(limit uint32) uint32 {
	if g_rand == nil {
		initRandSource()
	}
	return uint32(g_rand.Int31n(int32(limit)))
}

func GetRandFVal() float64 {
	if g_rand == nil {
		initRandSource()
	}
	return g_rand.Float64()
}

func Str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func Bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
