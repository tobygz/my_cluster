package parse

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type RpcSet struct {
	PkgName string
	Recv    string
	OutPath string

	funcParamsType map[string]func(string)

	RpcNameSlc []string
}

func newRpcSet() *RpcSet {
	rpc := &RpcSet{
		RpcNameSlc: make([]string, 0),
	}

	rpc.funcParamsType = map[string]func(string){
		"IRpcRequest": func(val string) {
			rpc.RpcNameSlc = append(rpc.RpcNameSlc, val)
		},
	}

	return rpc
}

func RpcFile(path string) (*RpcSet, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("os.Stat(%s) fail, err: %s", path, err)
	}

	rpc := newRpcSet()

	fileSet := token.NewFileSet()
	if info.IsDir() {
		return nil, fmt.Errorf("path %s not file", path)
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("os.Open(%s) fail, err: %s", path, err)
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("ioutil.ReadAll() fail, err: %s", err)
		}
		pFile, err := parser.ParseFile(fileSet, path, data, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parser.ParseFile(%s) fail, err: %s", path, err)
		}
		recognisedFuncByFile(rpc, pFile)

		rpc.PkgName = pFile.Name.Name
		rpc.OutPath = filepath.Join(filepath.Dir(path), "rpc_map.go")
	}

	//fmt.Println(chalk.Magenta.Color(fmt.Sprintf("%v", rpc)))
	return rpc, nil
}

func (s *RpcSet) SetPackage(val string) {
	s.PkgName = val
}

func (s *RpcSet) Receiver() string {
	return s.Recv
}

func (s *RpcSet) SetReceiver(val string) {
	s.Recv = val
}

func (s *RpcSet) MatchParamsType(paramsTp []string) (func(string), error) {
	var lastErr error
reqLoop:
	for reqStr, f := range s.funcParamsType {
		reqSlc := strings.Split(reqStr, ",")
		if len(paramsTp) != len(reqSlc) {
			lastErr = fmt.Errorf("params num %d invalid", len(paramsTp))
			continue reqLoop
		}

		for idx := 0; idx < len(reqSlc); idx++ {
			if paramsTp[idx] != reqSlc[idx] {
				lastErr = fmt.Errorf("param %s type invalid, need type %s", paramsTp[idx], reqSlc[idx])
				continue reqLoop
			}
		}

		return f, nil
	}

	return nil, lastErr
}
