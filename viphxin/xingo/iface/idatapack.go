package iface

type Idatapack interface {
	GetHeadLen() int32
	Unpack([]byte, interface{}) (interface{}, error)
	Pack(uint32, interface{}, []byte) ([]byte, error)
}
