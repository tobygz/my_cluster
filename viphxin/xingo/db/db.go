package db

import (
	"github.com/garyburd/redigo/redis"
	//"io"
	//"log"
	"strconv"
	"time"
)

var g_connect_timeout_sec int = 1
var g_read_timeout_sec int = 1
var g_write_timeout_sec int = 1

type Dbop struct {
	client redis.Conn
	MaxRid uint64
	isConn bool
}

//var Dbop *DbMgr = NewDbop()

func NewDbop() *Dbop {
	retObj := &Dbop{
		isConn: false,
	}
	return retObj
}

var g_dbid string
var g_addr string
var g_pwd string

func (this *Dbop) Start(dbid string, addr string, pwd string) {
	g_dbid = dbid
	g_addr = addr
	g_pwd = pwd

	client, err := redis.DialTimeout("tcp", addr, time.Duration(g_connect_timeout_sec)*time.Second,
		time.Duration(g_connect_timeout_sec)*time.Second, time.Duration(g_write_timeout_sec)*time.Second)
	if err != nil {
		panic(err)
		return
	}

	if pwd != "" {
		_, err = client.Do("AUTH", pwd)
		if err != nil {
			panic(err)
			client.Close()
			return
		}
	}

	_, err = client.Do("SELECT", dbid)
	if err != nil {
		panic(err)
		client.Close()
		return
	}
	this.client = client
	this.isConn = true
}

func (this *Dbop) DoDbTask(opCmd string, args ...interface{}) (interface{}, error) {
	return this.client.Do(opCmd, args...)
	/*
		ret, ok := this.client.Do(opCmd, args...)
		if ok != nil {
			log.Println("dodbtask :", ok, ",", ok.Error())
		}
		if ok != nil && ok.Error() == "EOF" {
			log.Println("DoDbTask reconn")
			this.Start(g_dbid, g_addr, g_pwd)
			//todo cache the cmd operate
			return "", nil
		}
		return ret, nil
	*/
}

func (this *Dbop) InitDb(val string) uint64 {
	nowKey := "login_max_rid"
	ret, err := this.DoDbTask("EXISTS", nowKey)
	if err != nil {
		panic(err)
		return uint64(0)
	}
	retb, err := redis.Bool(ret, nil)
	if retb == false {
		this.DoDbTask("SET", nowKey, val)
		retv, _ := strconv.ParseInt(val, 10, 64)
		return uint64(retv)
	} else {
		ret, err = this.DoDbTask("GET", nowKey)
		if err != nil {
			panic(err)
			return uint64(0)
		}
		retv, _ := redis.Uint64(ret, nil)
		return uint64(retv)
	}
	return uint64(0)
}
