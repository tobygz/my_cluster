package encry

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"path/filepath"
)

// RSA
// pub 公钥文件路径
// priv 私钥文件路径
func NewRsaCipher(pub, priv string) *RsaCipher {
	dir, err := filepath.Abs(filepath.Dir("."))
	if err != nil {
		panic("err")
	}
	data, err1 := ioutil.ReadFile(filepath.Join(dir, pub))
	if err1 != nil {
		panic(err1)
	}
	block, _ := pem.Decode(data)
	key, _ := x509.ParsePKIXPublicKey(block.Bytes)
	pubk := key.(*rsa.PublicKey)

	data, _ = ioutil.ReadFile(filepath.Join(dir, priv))
	block, _ = pem.Decode(data)
	privk, _ := x509.ParsePKCS1PrivateKey(block.Bytes)

	return &RsaCipher{
		pubk:  pubk,
		privk: privk,
	}
}

type RsaCipher struct {
	pubk  *rsa.PublicKey
	privk *rsa.PrivateKey
}

func (this *RsaCipher) Encrypt(src []byte) []byte {
	dst, err := rsa.EncryptOAEP(sha1.New(), rand.Reader, this.pubk, src, []byte(""))
	if err != nil {
		panic(err)
	}
	return dst
}

func (this *RsaCipher) Decrypt(src []byte) ([]byte, error) {
	dst, err := rsa.DecryptOAEP(sha1.New(), rand.Reader, this.privk, src, []byte(""))
	if err != nil {
		return nil, err
	}
	return dst, nil
}
