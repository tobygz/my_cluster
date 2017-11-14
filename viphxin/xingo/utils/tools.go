package utils

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"github.com/viphxin/xingo/logger"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
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
	if GlobalObject.LogFileType == logger.ROLLINGFILE {
		logger.SetRollingFile(GlobalObject.LogPath, GlobalObject.LogName,
			GlobalObject.MaxLogNum, GlobalObject.MaxFileSize, GlobalObject.LogFileUnit)
	} else {
		logger.SetRollingDaily(GlobalObject.LogPath, GlobalObject.LogName)
		logger.SetLevel(GlobalObject.LogLevel)
	}
	// --------------------------------------------init log end
}

func XingoTry(f reflect.Value, args []reflect.Value, handler func(interface{})) {
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
	f.Call(args)
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

func PrintStack() {
	debug.PrintStack()
}

var g_rand *rand.Rand

func GetRandVal(limit int) int {
	if g_rand == nil {
		g_rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return g_rand.Intn(limit)
}
