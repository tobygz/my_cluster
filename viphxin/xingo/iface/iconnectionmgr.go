package iface

type Iconnectionmgr interface {
	Add(Iconnection)
	FetchEncKey() []byte
	Remove(Iconnection) error
	Get(uint32) (Iconnection, error)
	Len() int
}
