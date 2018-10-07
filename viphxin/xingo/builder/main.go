package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/viphxin/xingo/builder/parse"
	"github.com/viphxin/xingo/builder/printer"
)

var (
	pb    = flag.String("pb", "", "protobuf descriptor file")
	api   = flag.String("api", "", "server api code path")
	rpc   = flag.String("rpc", "", "server rpc code path")
	debug = flag.Bool("debug", false, "output log")
)

func main() {
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if !*debug {
		log.SetOutput(ioutil.Discard)
	}

	as, err := parse.ApiFile(*api)
	if err != nil {
		log.Println("parse ApiFile:", err)
		return
	} else {
		if as.PbSet, err = parse.DescriptorSet(*pb); err != nil {
			log.Println("parse DescriptorSet:", err)
			return
		}
	}

	rs, err := parse.RpcFile(*rpc)
	if err != nil {
		log.Println("parse RpcFile:", err)
		return
	}

	printer.NewPrinter(as, rs).Run()

	return
}
