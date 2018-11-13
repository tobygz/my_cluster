package logger

import (
	log "github.com/golang/glog"
)

const (
	_VER string = "1.0.2"
)

type UNIT uint64

const (
	_       = iota
	KB UNIT = 1 << (iota * 10)
	MB
	GB
	TB
)

type LEVEL int32

const (
	ALL LEVEL = iota
	TRACE
	DEBUG
	INFO
	PROF
	WARN
	ERROR
	FATAL
	OFF
)

const (
	_ = iota
	ROLLINGDAILY
	ROLLINGFILE
)

func SetPrefix(title string) {
}

func SetFileMaxSize(maxSize int64, _unit UNIT) {
	log.MaxSize = uint64(maxSize) * uint64(_unit)
}

func SetConsole(b bool) {
	log.SetAlsoToStderr(b)
}

func SetToSyslog(b bool) {
	log.SetToSyslog(b)
}

func SetSyslogAddr(addr string, port int) {
	if addr != "" && port > 0 {
		log.SetSyslogAddr(addr, port)
	}
}

func SetLevel(_level LEVEL) {
	if _level <= INFO {
		log.SetVerbosity("INFO")
	} else if _level == WARN {
		log.SetVerbosity("WARNING")
	} else if _level == ERROR {
		log.SetVerbosity("ERROR")
	} else {
		log.SetVerbosity("FATAL")
	}
}

func SetLogFileLine(b bool) {
	log.SetLogFileLine(b)
}

func SetRollingFile(fileDir, fileName string) {
	log.SetLogPath(fileDir, fileName)
	log.ResetOutput()
}

func SetRollingDaily(fileDir, fileName string) {
	log.SetLogPath(fileDir, fileName)
	log.ResetOutput()
}

func Flush() {
	log.Flush()
}

var (
	Trace logLnFunc = log.Infoln
	Debug logLnFunc = log.Infoln
	Info  logLnFunc = log.Infoln
	Prof  logLnFunc = log.Warningln
	Warn  logLnFunc = log.Warningln
	Error logLnFunc = log.Errorln
	Fatal logLnFunc = log.Fatalln

	Infof  logfFunc = log.Infof
	Warnf  logfFunc = log.Warningf
	Errorf logfFunc = log.Errorf
	Fatalf logfFunc = log.Fatalf
)

type logLnFunc func(v ...interface{})
type logfFunc func(format string, v ...interface{})
