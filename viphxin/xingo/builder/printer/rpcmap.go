package printer

import (
	"bytes"
	"fmt"
)

type rpcGen struct {
	w *bytes.Buffer
}

func rpc(w *bytes.Buffer) *rpcGen {
	return &rpcGen{
		w: w,
	}
}

func (p *rpcGen) Execute(pr *Printer) {
	p.w.WriteString(fmt.Sprintf("package %s\n", pr.rs.PkgName))

	//GetRpcMap
	p.w.WriteString(fmt.Sprintf("func (this *%s) GetRpcMap() map[string]func(iface.IRpcRequest) {\n", pr.rs.Recv))
	p.w.WriteString("return map[string]func(iface.IRpcRequest) {\n")
	for _, name := range pr.rs.RpcNameSlc {
		p.w.WriteString(fmt.Sprintf("\"%s\": this.%s,\n", name, name))
	}
	p.w.WriteString("}\n}\n")
}
