package web

import (
	json "github.com/json-iterator/go"
	"net/url"
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
		k := strings.ToLower(ary2[0])
		if u, err := url.QueryUnescape(ary2[1]); err != nil {
			(*ret)[k] = ary2[1]
		} else {
			(*ret)[k] = u
		}
	}

	return ret
}

func (c *CgiInput) ToString() string {
	ret, _ := json.Marshal(c)
	return string(ret)
}

func (c *CgiInput) Exist(k string) bool {
	key := strings.ToLower(k)
	if _, ok := (*c)[key]; ok {
		return true
	}
	return false
}

func (c *CgiInput) String(k string) string {
	key := strings.ToLower(k)
	if v, ok := (*c)[key]; ok {
		return v
	}
	return ""
}

func (c *CgiInput) Int(k string) int {
	key := strings.ToLower(k)
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.Atoi(v)
		return u
	}
	return 0
}

func (c *CgiInput) Uint32(k string) uint32 {
	key := strings.ToLower(k)
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.ParseUint(v, 10, 32)
		return uint32(u)
	}
	return 0
}

func (c *CgiInput) Uint64(k string) uint64 {
	key := strings.ToLower(k)
	if v, ok := (*c)[key]; ok {
		u, _ := strconv.ParseUint(v, 10, 64)
		return u
	}
	return 0
}

func (c *CgiInput) Bool(k string) bool {
	key := strings.ToLower(k)
	if v, ok := (*c)[key]; ok {
		switch v {
		case "true", "yes", "T", "Y":
			return true
		}
	}
	return false
}
