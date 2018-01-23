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
	return
}

func SetConsole(isConsole bool) {
	log.SetAlsoToStderr(isConsole)
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

func SetRollingFile(fileDir, fileName string, maxNumber int32, maxSize int64, _unit UNIT) {
	log.SetLogPath(fileDir, fileName)
	log.MaxSize = uint64(maxSize) * uint64(_unit)
	log.ResetOutput()
}

func SetRollingDaily(fileDir, fileName string) {
	log.SetLogPath(fileDir, fileName)
	log.ResetOutput()
}

var (
	Trace logLnFunc = log.Infoln
	Debug logLnFunc = log.Infoln
	Info  logLnFunc = log.Infoln
	Prof  logLnFunc = log.Warningln
	Warn  logLnFunc = log.Warningln
	Error logLnFunc = log.Errorln
	Fatal logLnFunc = log.Fatalln
)

type logLnFunc func(v ...interface{})

/*
func init() {
	Trace = log.Infoln
	Debug = log.Infoln
	Info = log.Infoln
	Warn = log.Warningln
	Error = log.Errorln
	Fatal = log.Fatalln
}
*/
