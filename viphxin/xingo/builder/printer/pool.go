package printer

import (
	"bytes"
	"fmt"
)

type poolGen struct {
	w *bytes.Buffer
}

func pool(w *bytes.Buffer) *poolGen {
	return &poolGen{
		w: w,
	}
}

func (p *poolGen) Execute(pr *Printer) {
	p.w.WriteString("package pool\n")

	matchPbNum := 0
	p.w.WriteString("var (\n")
	for msgId, _ := range pr.as.ApiId2Name {
		if _, ok := pr.as.PbSet.MsgId2Name[msgId]; ok {
			matchPbNum = matchPbNum + 1
			p.w.WriteString(fmt.Sprintf("Pool_%d sync.Pool\n", msgId))
		}
	}
	p.w.WriteString(")\n")

	if matchPbNum <= 0 {
		p.w.Reset()
		return
	}

	p.w.WriteString("func init() {\n")
	for msgId, _ := range pr.as.ApiId2Name {
		if name, ok := pr.as.PbSet.MsgId2Name[msgId]; ok {
			p.w.WriteString(fmt.Sprintf("Pool_%d.New = func() interface{} {\nreturn &pb.%s{}\n}\n", msgId, name))
		}
	}
	p.w.WriteString("}\n")
}
