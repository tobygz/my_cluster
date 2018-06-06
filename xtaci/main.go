package main

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"github.com/xtaci/kcp-go"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

const portEcho = "0.0.0.0:20020"
const portSink = "127.0.0.1:19999"
const portTinyBufferEcho = "127.0.0.1:29999"
const portListerner = "127.0.0.1:9998"
const salt = "kcptest"

var key = []byte("testkey")
var fec = 4
var pass = pbkdf2.Key(key, []byte(portSink), 4096, 32, sha1.New)

func main() {
	fg := flag.Bool("s", false, "")
	flag.Parse()

	if *fg {
		log.Println("beginning server, port:", portEcho)
		go echoServer()
	} else {
		log.Println("beginning client, ")
		go TestSendRecv()
	}

	ch := make(chan int, 1)
	<-ch

}

func dialEcho() (*kcp.UDPSession, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	//block, _ := kcp.NewSalsa20BlockCrypt(pass)
	sess, err := kcp.DialWithOptions(portEcho, nil, 10, 3)
	if err != nil {
		panic(err)
	}

	sess.SetStreamMode(true)
	sess.SetStreamMode(false)
	sess.SetStreamMode(true)
	sess.SetWindowSize(4096, 4096)
	sess.SetReadBuffer(4 * 1024 * 1024)
	sess.SetWriteBuffer(4 * 1024 * 1024)
	sess.SetStreamMode(true)
	sess.SetNoDelay(1, 10, 2, 1)
	sess.SetMtu(1400)
	sess.SetMtu(1600)
	sess.SetMtu(1400)
	sess.SetACKNoDelay(true)
	sess.SetDeadline(time.Now().Add(time.Minute))
	return sess, err
}

//////////////////////////
func listenEcho() (net.Listener, error) {
	//block, _ := NewNoneBlockCrypt(pass)
	//block, _ := NewSimpleXORBlockCrypt(pass)
	//block, _ := NewTEABlockCrypt(pass[:16])
	//block, _ := NewAESBlockCrypt(pass)
	//block, _ := kcp.NewSalsa20BlockCrypt(pass)
	return kcp.ListenWithOptions(portEcho, nil, 10, 3)
}

func echoServer() {
	l, err := listenEcho()
	if err != nil {
		panic(err)
	}

	go func() {
		kcplistener := l.(*kcp.Listener)
		kcplistener.SetReadBuffer(4 * 1024 * 1024)
		kcplistener.SetWriteBuffer(4 * 1024 * 1024)
		kcplistener.SetDSCP(46)
		for {
			s, err := l.Accept()
			if err != nil {
				return
			}

			log.Println("got conn:")
			// coverage test
			s.(*kcp.UDPSession).SetReadBuffer(4 * 1024 * 1024)
			s.(*kcp.UDPSession).SetWriteBuffer(4 * 1024 * 1024)
			go handleEcho(s.(*kcp.UDPSession))
		}
	}()
}

///////////////////////////

func handleEcho(conn *kcp.UDPSession) {
	conn.SetStreamMode(true)
	conn.SetWindowSize(4096, 4096)
	conn.SetNoDelay(1, 10, 2, 1)
	conn.SetDSCP(46)
	conn.SetMtu(1400)
	conn.SetACKNoDelay(false)
	conn.SetReadDeadline(time.Now().Add(time.Hour))
	conn.SetWriteDeadline(time.Now().Add(time.Hour))
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			panic(err)
		}
		log.Println("got ", n, " bytes buf:", conn.RemoteAddr())
		conn.Write(buf[:n])
	}
}

///////////////////////////
func TestSendRecv() {
	cli, err := dialEcho()
	if err != nil {
		panic(err)
	}
	log.Println("connect serv succ")
	cli.SetWriteDelay(true)
	cli.SetDUP(1)
	const N = 100
	buf := make([]byte, 10)
	for i := 0; i < N; i++ {
		msg := fmt.Sprintf("hello%v", i)
		cli.Write([]byte(msg))
		if n, err := cli.Read(buf); err == nil {
			if string(buf[:n]) != msg {
				log.Println("fail here")
			}
			log.Println("recv msg:", buf[:n])
		} else {
			panic(err)
		}

		time.Sleep(2 * time.Second)
	}
	cli.Close()
}

func echo_tester(cli net.Conn, msglen, msgcount int) error {
	buf := make([]byte, msglen)
	for i := 0; i < msgcount; i++ {
		// send packet
		if _, err := cli.Write(buf); err != nil {
			return err
		}

		// receive packet
		nrecv := 0
		for {
			n, err := cli.Read(buf)
			if err != nil {
				return err
			} else {
				nrecv += n
				if nrecv == msglen {
					break
				}
			}
		}
	}
	return nil
}
