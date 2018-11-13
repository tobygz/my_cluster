package parse

import (
	"go/ast"
	"log"
)

type ServerModule interface {
	SetPackage(string)
	Receiver() string
	SetReceiver(string)

	MatchParamsType([]string) (func(string), error)
}

func recognisedFuncByFile(m ServerModule, file *ast.File) {
	paramsTp := make([]string, 0)
funcLoop:
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			name := decl.Name.Name
			paramsTp = paramsTp[:0]
			params := decl.Type.Params.List
			for _, param := range params {
				var typeStr string
				switch paramType := param.Type.(type) {
				case *ast.SelectorExpr:
					typeStr = paramType.Sel.Name
				case *ast.Ident:
					typeStr = paramType.Name
				default:
					log.Printf("func %s param %v type unknown", name, param.Names)
					continue funcLoop
				}

				for _ = range param.Names {
					paramsTp = append(paramsTp, typeStr)
				}
			} //for _, param := range params

			addF, err := m.MatchParamsType(paramsTp)
			if addF == nil || err != nil {
				log.Printf("func %s params %v not match, return %s", name, params, err)
				continue
			}

			if !chkRecvAndSet(m, decl.Recv) {
				log.Printf("func %s recv %v invalid", name, decl.Recv.List)
				continue
			}

			addF(name)
		} //switch decl := decl.(type)
	} //for _, decl := range file.Decls
}

func chkRecvAndSet(m ServerModule, recv *ast.FieldList) bool {
	if len(recv.List) != 1 {
		return false
	}
	var typeStr string
	switch tp := recv.List[0].Type.(type) {
	case *ast.StarExpr:
		switch x := tp.X.(type) {
		case *ast.Ident:
			typeStr = x.Name
		default:
			return false
		}
	default:
		return false
	}

	if m.Receiver() == "" {
		m.SetReceiver(typeStr)
	} else if m.Receiver() != typeStr {
		return false
	}
	return true
}
