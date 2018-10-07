package web

import (
	"strconv"
	"strings"
)

type CgiInput map[string]string

func parseReqBody(reqBody string) *CgiInput {
	ret := new(CgiInput)
	*ret = CgiInput(make(map[string]string, 0))
	ary0 := strings.Split(reqBody, "?")
	(*ret)["innerreqname"] = ary0[0]
	if len(ary0) < 2 {
		return ret
	}

	//ary0[1]  a=3&b=2
	ary1 := strings.Split(ary0[1], "&")
	if len(ary1) == 0 {
		return ret
	}
	//ary1 [a=3],[b=2]
	for _, elem := range ary1 {
		ary2 := strings.Split(elem, "=")
		if len(ary2) != 2 {
			continue
		}
		(*ret)[ary2[0]] = ary2[1]
	}

	return ret
}

func (c *CgiInput) Exist(key string) bool {
	if _, ok := (*c)[key]; ok {
		return true
	}
	return false
}

func (c *CgiInput) String(key string) string {
	if v, ok := (*c)[key]; ok {
		return v
	}
	return ""
}

func (c *CgiInput) Int(key string) int {
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.Atoi(v)
		return u
	}
	return 0
}

func (c *CgiInput) Uint32(key string) uint32 {
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.ParseUint(v, 10, 32)
		return uint32(u)
	}
	return 0
}

func (c *CgiInput) Uint64(key string) uint64 {
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.ParseUint(v, 10, 64)
		return u
	}
	return 0
}

func (c *CgiInput) Bool(key string) bool {
	if v, ok := (*c)[key]; ok {
		switch v {
		case "true", "yes", "T", "Y":
			return true
		}
	}
	return false
}
