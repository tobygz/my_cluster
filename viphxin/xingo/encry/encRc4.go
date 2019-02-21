package encry

import (
	"strconv"
)

// A Cipher is an instance of RC4 using a particular key.
type Cipher struct {
	S    [256]uint32
	S_bk [256]uint32
	i, j uint8
}

type KeySizeError int

func (k KeySizeError) Error() string {
	return "crypto/rc4: invalid key size " + strconv.Itoa(int(k))
}

// NewCipher creates and returns a new Cipher. The key argument should be the
// RC4 key, at least 1 byte and at most 256 bytes.
func NewCipher(key []byte) (*Cipher, error) {
	k := len(key)
	if k < 1 || k > 256 {
		return nil, KeySizeError(k)
	}
	var c Cipher
	for i := 0; i < 256; i++ {
		c.S[i] = uint32(i)
	}
	var j uint8 = 0
	for i := 0; i < 256; i++ {
		j += uint8(c.S[i]) + key[i%k]
		c.S[i], c.S[j] = c.S[j], c.S[i]
	}

	//do backup
	for ii := range c.S {
		c.S_bk[ii] = c.S[ii]
	}
	return &c, nil
}

// Reset zeros the key data so that it will no longer appear in the
// process's memory.
func (c *Cipher) Reset() {
	for i := range c.S {
		c.S[i] = 0
	}
	for i := range c.S_bk {
		c.S[i] = c.S_bk[i]
	}
	c.i, c.j = 0, 0
}

// xorKeyStreamGeneric sets dst to the result of XORing src with the
// key stream. Dst and src must overlap entirely or not at all.
//
// This is the pure Go version. rc4_{amd64,386,arm}* contain assembly
// implementations. This is here for tests and to prevent bitrot.
func (c *Cipher) XorKeyStreamGeneric(dst, src []byte) {
	if len(src) == 0 {
		return
	}
	c.Reset()

	i, j := c.i, c.j
	_ = dst[len(src)-1]
	dst = dst[:len(src)] // eliminate bounds check from loop
	for k, v := range src {
		i += 1
		x := c.S[i]
		j += uint8(x)
		y := c.S[j]
		c.S[i], c.S[j] = y, x
		dst[k] = v ^ uint8(c.S[uint8(x+y)])
	}
	c.i, c.j = i, j
}
