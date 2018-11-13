package parse

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ApiSet struct {
	PkgName string
	Recv    string
	OutPath string

	ApiNameSlc   []string
	ApiName64Slc []string
	ApiId2Name   map[uint32]string

	funcParamsType map[string]func(string)

	PbSet *ProtobufSet
}

func newApiSet() *ApiSet {
	api := &ApiSet{
		ApiNameSlc:   make([]string, 0),
		ApiName64Slc: make([]string, 0),
		ApiId2Name:   make(map[uint32]string, 0),
	}

	api.funcParamsType = map[string]func(string){
		"IRequest,uint32,uint32": func(val string) {
			api.ApiNameSlc = append(api.ApiNameSlc, val)

			if msgId := getMsgIdByName(val); msgId > 0 {
				api.ApiId2Name[msgId] = val
			}
		},
		"IRequest,uint32,uint64": func(val string) {
			api.ApiName64Slc = append(api.ApiName64Slc, val)

			if msgId := getMsgIdByName(val); msgId > 0 {
				api.ApiId2Name[msgId] = val
			}
		},
	}

	return api
}

func ApiFile(path string) (*ApiSet, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("os.Stat(%s) fail, err: %s", path, err)
	}

	api := newApiSet()

	fileSet := token.NewFileSet()
	if info.IsDir() {
		pkgMap, err := parser.ParseDir(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("parser.ParseDir(%s) fail, err: %s", path, err)
		}
		if len(pkgMap) != 1 {
			return nil, fmt.Errorf("%s has multiple package", path)
		}
		for _, pkg := range pkgMap {
			api.PkgName = pkg.Name
			for _, file := range pkg.Files {
				recognisedFuncByFile(api, file)
			}
		}

		api.OutPath = filepath.Join(path, "api_map.go")
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
		recognisedFuncByFile(api, pFile)

		api.PkgName = pFile.Name.Name
		api.OutPath = filepath.Join(filepath.Dir(path), "api_map.go")
	}

	//fmt.Println(chalk.Magenta.Color(fmt.Sprintf("%v", api)))
	return api, nil
}

func (s *ApiSet) SetPackage(val string) {
	s.PkgName = val
}

func (s *ApiSet) Receiver() string {
	return s.Recv
}

func (s *ApiSet) SetReceiver(val string) {
	s.Recv = val
}

func (s *ApiSet) MatchParamsType(paramsTp []string) (func(string), error) {
	var lastErr error
reqLoop:
	for reqStr, f := range s.funcParamsType {
		reqSlc := strings.Split(reqStr, ",")
		if len(paramsTp) != len(reqSlc) {
			lastErr = fmt.Errorf("params num %d invalid", len(paramsTp))
			continue reqLoop
		}

		for idx := 0; idx < len(reqSlc); idx++ {
			if ok, err := regexp.MatchString(paramsTp[idx], reqSlc[idx]); !ok || err != nil {
				lastErr = fmt.Errorf("param %s type invalid, need type %s", paramsTp[idx], reqSlc[idx])
				continue reqLoop
			}
		}

		return f, nil
	}

	return nil, lastErr
}
