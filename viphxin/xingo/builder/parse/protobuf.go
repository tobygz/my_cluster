package parse

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type ProtobufSet struct {
	PkgName      string
	MsgId2Name   map[uint32]string
	EmptyMsgName []string
}

func DescriptorSet(descFile string) (*ProtobufSet, error) {
	f, err := os.Open(descFile)
	if err != nil {
		return nil, fmt.Errorf("opening file fail, err: %v", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading input fail, err: %v", err)
	}

	descSet := new(descriptor.FileDescriptorSet)
	if err := proto.Unmarshal(data, descSet); err != nil {
		return nil, fmt.Errorf("unmarshal serial fail, err: %v", err)
	}

	pbSet := recognisedMsg(descSet)
	//fmt.Println(chalk.Magenta.Color(fmt.Sprintf("%v", pbSet)))

	return pbSet, nil
}

func getMsgIdByName(msgName string) uint32 {
	msgPSlc := strings.Split(msgName, "_")

	length := len(msgPSlc)
	if length <= 0 {
		return 0
	}

	msgId, err := strconv.ParseUint(msgPSlc[length-1], 10, 32)
	if err != nil {
		log.Println("strconv.ParseUint return", err)
		return 0
	}

	return uint32(msgId)
}

func recognisedMsg(descSet *descriptor.FileDescriptorSet) *ProtobufSet {
	pbSet := &ProtobufSet{
		MsgId2Name:   make(map[uint32]string, 0),
		EmptyMsgName: make([]string, 0),
	}
	for _, file := range descSet.GetFile() {
		if pbSet.PkgName == "" {
			pbSet.PkgName = file.GetPackage()
		} else if pbSet.PkgName != file.GetPackage() {
			continue
		}

		for _, msg := range file.GetMessageType() {
			if !strings.Contains(msg.GetName(), "C2S") {
				continue
			}

			msgId := getMsgIdByName(msg.GetName())
			if msgId <= 0 {
				continue
			}

			emptyMsg := &descriptor.DescriptorProto{Name: msg.Name}
			if proto.Equal(emptyMsg, msg) {
				pbSet.EmptyMsgName = append(pbSet.EmptyMsgName, msg.GetName())
				continue
			}

			pbSet.MsgId2Name[msgId] = msg.GetName()
		}
	}

	return pbSet
}
