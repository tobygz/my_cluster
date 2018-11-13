package printer

import (
	"bytes"
	"fmt"
)

type apiGen struct {
	w *bytes.Buffer
}

func api(w *bytes.Buffer) *apiGen {
	return &apiGen{
		w: w,
	}
}

func (p *apiGen) Execute(pr *Printer) {
	p.w.WriteString(fmt.Sprintf("package %s\n", pr.as.PkgName))

	//GetApiMap
	p.w.WriteString(fmt.Sprintf("func (this *%s) GetApiMap() map[string]func(iface.IRequest, uint32, uint32) {\n", pr.as.Recv))
	if len(pr.as.ApiNameSlc) > 0 {
		p.w.WriteString("return map[string]func(iface.IRequest, uint32, uint32) {\n")
		for _, name := range pr.as.ApiNameSlc {
			p.w.WriteString(fmt.Sprintf("\"%s\": this.%s,\n", name, name))
		}
		p.w.WriteString("}\n")
	} else {
		p.w.WriteString("return nil\n")
	}
	p.w.WriteString("}\n\n")

	//GetGApiMap
	p.w.WriteString(fmt.Sprintf("func (this *%s) GetGApiMap() map[string]func(iface.IRequest, uint32, uint64) {\n", pr.as.Recv))
	if len(pr.as.ApiName64Slc) > 0 {
		p.w.WriteString("return map[string]func(iface.IRequest, uint32, uint64) {\n")
		for _, name := range pr.as.ApiName64Slc {
			p.w.WriteString(fmt.Sprintf("\"%s\": this.%s,\n", name, name))
		}
		p.w.WriteString("}\n")
	} else {
		p.w.WriteString("return nil\n")
	}
	p.w.WriteString("}\n\n")

	//UnmarshalProtMsg
	matchPbNum := 0
	poolBuff := bytes.NewBuffer(nil)
	poolBuff.WriteString("func UnmarshalProtMsg(inter interface{}) {\n")
	poolBuff.WriteString("bobj := inter.(*fnet.PkgData)\n")
	poolBuff.WriteString("var msg proto.Message\n")
	poolBuff.WriteString("switch bobj.MsgId {\n")
	for msgId, _ := range pr.as.ApiId2Name {
		if name, ok := pr.as.PbSet.MsgId2Name[msgId]; ok {
			matchPbNum = matchPbNum + 1
			poolBuff.WriteString(fmt.Sprintf("case %d:\nmsg = pool.Pool_%d.Get().(*pb.%s)\n", msgId, msgId, name))
		}
	}
	poolBuff.WriteString("}\nif msg == nil {\nreturn\n}\n")
	poolBuff.WriteString("proto.Unmarshal(bobj.Data, msg)\nbobj.SetMsgObj(msg)\n}\n")
	if matchPbNum > 0 {
		p.w.Write(poolBuff.Bytes())
	}
}
