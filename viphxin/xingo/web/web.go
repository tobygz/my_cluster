package web

import (
	"fmt"
	"github.com/viphxin/xingo/logger"
	"net"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"strings"
)

type DataReq struct {
	reqStr string
	conn   net.Conn
}

type HandleReq struct {
	name   string
	reqStr string
}

type Web struct {
	reqChan chan interface{}
	apis    map[string]reflect.Value
	l       net.Listener
	run     bool
}

var GlobalWeb *Web

func NewWeb() *Web {
	if GlobalWeb != nil {
		return GlobalWeb
	}
	GlobalWeb := &Web{
		reqChan: make(chan interface{}, 16),
		apis:    make(map[string]reflect.Value, 32),
		run:     true,
	}

	return GlobalWeb
}

func (this *Web) GetReqChan() chan interface{} {
	return this.reqChan
}

func (this *Web) HandleReqCall(re interface{}) {
	req := re.(*DataReq)
	this.handleRequest(req.conn, req.reqStr)
}

//for admin only, called in gate is forbidden
func (this *Web) StartParseReq() {
	if this.run == false {
		return
	}
	for {
		re := <-this.reqChan
		if re == nil {
			break
		}
		req := re.(*DataReq)
		if req != nil {
			this.handleRequest(req.conn, req.reqStr)
		}
	}
}

func (this *Web) Start(port string) {
	port = fmt.Sprintf(":%s", port)
	var err error
	this.l, err = net.Listen("tcp", port)
	if err != nil {
		logger.Debug("Error listening:", err)
		return
	}
	defer this.l.Close()
	logger.Debug("Listening on " + ":" + port)
	counter := 0
	for {
		counter++
		logger.Debug("ready accept :", counter)
		conn, err := this.l.Accept()
		if err != nil {
			logger.Debug("Error accepting: ", err)
			return
		}
		//logs an incoming message
		//fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.

		go func(ct int) {
			defer func() {
				if err := recover(); err != nil {
					logger.Info("-------------panic recover---------------")
					buf := make([]byte, 1<<16)
					stackSize := runtime.Stack(buf, true)
					logger.Error(fmt.Sprintf("%s\n", string(buf[0:stackSize])))
				}
			}()
			//logger.Debug("ready read: ", ct)
			strTotal := ""
			for {
				tbuf := make([]byte, 4096)
				n, err := conn.Read(tbuf)
				if err != nil {
					logger.Debug("Error to read message because of ", err)
					conn.Close()
					break
				}
				strTotal = fmt.Sprintf("%s%s", strTotal, string(tbuf[:n]))

				//logger.Debug("ready after read: ", ct)
				if ok, retAry := checkFullReq(strTotal); ok {
					tmpReqStr := parseReq(retAry)
					if tmpReqStr == "" {
						return
					}
					this.reqChan <- &DataReq{
						reqStr: tmpReqStr,
						conn:   conn,
					}
					logger.Debug("append len:", len(this.reqChan), ", str:", tmpReqStr, ",ct:", ct)
					break
				} else {
					logger.Debug("buf: %s not full, ct", string(tbuf[:n]), ct)
				}
			}
		}(counter)
	}
	logger.Info("web thread exited...")
}

func (this *Web) AddHandles(prefix string, router interface{}) {
	value := reflect.ValueOf(router)
	tp := value.Type()
	for i := 0; i < value.NumMethod(); i++ {
		name := tp.Method(i).Name
		name = fmt.Sprintf("%s/%s", prefix, strings.ToLower(name))
		logger.Info("http AddHandles add ", name)
		this.apis[name] = value.Method(i)
	}
}

func parseReqBody(reqBody string) *map[string]string {
	ret := make(map[string]string, 0)
	ary0 := strings.Split(reqBody, "?")
	ret["innerreqname"] = ary0[0]
	if len(ary0) < 2 {
		return &ret
	}

	//ary0[1]  a=3&b=2
	ary1 := strings.Split(ary0[1], "&")
	if len(ary1) == 0 {
		return &ret
	}
	//ary1 [a=3],[b=2]
	for _, elem := range ary1 {
		ary2 := strings.Split(elem, "=")
		if len(ary2) != 2 {
			continue
		}
		ret[ary2[0]] = ary2[1]
	}

	return &ret
}

func (this *Web) handleRequest(conn net.Conn, reqBody string) {
	defer conn.Close()

	parseMap := parseReqBody(reqBody)
	reqName, _ := (*parseMap)["innerreqname"]
	reqName = strings.ToLower(reqName)
	//valSend := fmt.Sprintf("hello Web golang reqName: %s, map length: %d", reqName, len(*parseMap))

	logger.Debug("handleRequest req: ", reqName)
	var valSend string
	f, ok := this.apis[reqName]
	if !ok {
		valSend = fmt.Sprintf("req:%s not found", reqName)
		logger.Debug("handleRequest:", valSend)
	} else {
		tmpret := f.Call([]reflect.Value{reflect.ValueOf(parseMap)})
		valSend = tmpret[0].String()
	}

	var contentTp string
	if strings.HasPrefix(reqName, "/json") {
		contentTp = "application/json"
	} else {
		contentTp = "text/html"
		valSend = fmt.Sprintf("<html><head></head><body>%s</body></html>", valSend)
	}
	sendByte := fmt.Sprintf("HTTP/1.0 200 OK\r\nContent-Type:%s;charset=utf-8\r\nContent-Length:%d\r\n\r\n%s", contentTp, len(valSend), valSend)
	conn.Write([]byte(sendByte))
	conn.Close()
}

func (this *Web) RawClose() {
	this.l.Close()
	close(this.reqChan)
	this.run = false
}

func (this *Web) WaitSignal() {
	// close
	c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt, os.Kill)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	this.RawClose()
	logger.Info("Web close sig: ", sig)
}

func parseReqAry(reqStr string) []string {
	retAry := strings.Split(reqStr, "\r\n")
	nowLen := len(retAry)
	retAry[nowLen-1] = strings.Trim(retAry[nowLen-1], string([]byte{0}))
	return retAry
}

func checkFullReq(reqStr string) (bool, []string) {
	retAry := parseReqAry(reqStr)
	nowLen := len(retAry)
	if nowLen < 2 {
		return false, retAry
	}

	if retAry[nowLen-1] == "" && retAry[nowLen-2] == "" {
		return true, retAry
	}

	return false, retAry
}

func parseReq(reqAry []string) string {
	retAry := strings.Split(reqAry[0], " ")
	if len(retAry) < 2 {
		return ""
	}
	return retAry[1]
}

/*
//eg: handle
type Handle struct {
}

func (this *Handle) Test(mapParam *map[string]string) string {
        return fmt.Sprintf("called in test len: %d", len(*mapParam))
}

func main() {
        Webobj := NewWeb()
        Webobj.AddHandles(&Handle{})
        Webobj.Start(":8080")

        Webobj.WaitSignal()
}
*/
