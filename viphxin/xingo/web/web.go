package web

import (
	"fmt"
	"github.com/viphxin/xingo/logger"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"time"
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
	reqChan chan *DataReq
	apis    map[string]reflect.Value
}

var GlobalWeb *Web

func NewWeb() *Web {
	if GlobalWeb != nil {
		return GlobalWeb
	}
	GlobalWeb := &Web{
		reqChan: make(chan *DataReq, 16),
		apis:    make(map[string]reflect.Value, 32),
	}

	return GlobalWeb
}

func (this *Web) StartParseReq() {
	for {
		select {
		case req := <-this.reqChan:
			this.handleRequest(req.conn, req.reqStr)
		case <-time.After(time.Microsecond * 0):
			return
		}
	}
}

func (this *Web) Start(port string) {
	port = fmt.Sprintf(":%s", port)
	var l net.Listener
	var err error
	l, err = net.Listen("tcp", port)
	if err != nil {
		logger.Debug("Error listening:", err)
		return
	}
	defer l.Close()
	this.StartParseReq()
	logger.Debug("Listening on " + ":" + port)
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Debug("Error accepting: ", err)
			return
		}
		//logs an incoming message
		fmt.Printf("Received message %s -> %s \n", conn.RemoteAddr(), conn.LocalAddr())
		// Handle connections in a new goroutine.

		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			logger.Debug("Error to read message because of ", err)
			continue
		}
		if ok, retAry := checkFullReq(string(buf)); ok {
			tmpReqStr := parseReq(retAry)
			this.reqChan <- &DataReq{
				reqStr: tmpReqStr,
				conn:   conn,
			}
			logger.Debug("append len:", len(this.reqChan), ", str:", tmpReqStr)
		} else {
			logger.Debug("buf: %s not full", string(buf))
		}
	}
}

func (this *Web) AddHandles(router interface{}) {
	value := reflect.ValueOf(router)
	tp := value.Type()
	for i := 0; i < value.NumMethod(); i++ {
		name := tp.Method(i).Name
		name = fmt.Sprintf("/%s", strings.ToLower(name))
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

	valSend = fmt.Sprintf("<html><head></head><body>%s</body></html>", valSend)
	sendByte := fmt.Sprintf("HTTP/1.0 200 OK\r\nContent-Type:text/html;charset=utf-8\r\nContent-Length:%d\r\n\r\n%s", len(valSend), valSend)
	conn.Write([]byte(sendByte))
	conn.Close()
}

func (this *Web) WaitSignal() {
	// close
	c := make(chan os.Signal, 1)
	//signal.Notify(c, os.Interrupt, os.Kill)
	signal.Notify(c, os.Interrupt, os.Kill)
	sig := <-c
	logger.Debug(sig)
	close(this.reqChan)
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
