package encry

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"fmt"
	"log"
	mrd "math/rand"
)

var CHANGE_SALT_NUM int = 300

type AesAlg struct {
	Salt     []byte
	Key      []byte
	Iv       []byte
	Block    cipher.Block
	Cbc      cipher.BlockMode //for decrypt
	Ebc      cipher.BlockMode //for encrypt
	Num      int
	ConstKey string
}

func NewAesAlg(pwd []byte) *AesAlg {
	obj := &AesAlg{}
	obj.Num = 0
	obj.Init(pwd)
	return obj
}

func (this *AesAlg) Init(pwd []byte) {
	this.ConstKey = "Ym"
	this.Salt = make([]byte, 4)
	for i := 0; i < 4; i++ {
		if i > len(pwd)-1 {
			this.Salt[i] = 0
		} else {
			this.Salt[i] = pwd[i]
		}
	}
	passphrase := string(pwd)
	this.Key, this.Iv = __DeriveKeyAndIv(passphrase, string(this.Salt))
	log.Println("aes alg init, salt:", this.Salt, ", key:", this.Key, ", Iv:", this.Iv)

	var err error
	this.Block, err = aes.NewCipher(this.Key)
	if err != nil {
		panic(err)
	}
	this.Cbc = cipher.NewCBCDecrypter(this.Block, this.Iv)
	this.Ebc = cipher.NewCBCEncrypter(this.Block, this.Iv)
}

// Encrypts text with the passphrase
func (this *AesAlg) Encrypt(text []byte) []byte {

	pad := __PKCS5Padding([]byte(text), this.Block.BlockSize())
	encrypted := make([]byte, len(pad))
	this.Ebc.CryptBlocks(encrypted, pad)

	flag := string([]byte{0})
	this.Num++
	if this.Num >= CHANGE_SALT_NUM {
		this.Num = 0
		flag = string([]byte{1})
		newPwd := []byte(fmt.Sprintf("%v", RandUint(1000, 9999)))
		this.Init(newPwd)
	}

	return []byte(this.ConstKey + flag + string(this.Salt) + string(encrypted))
}

// Decrypts encrypted text with the passphrase
func (this *AesAlg) Decrypt(ct []byte) []byte {
	constLen := len(this.ConstKey) + len(this.Salt)
	if len(ct) < constLen || string(ct[:len(this.ConstKey)]) != this.ConstKey {
		return nil
	}

	ct = ct[constLen:]

	dst := make([]byte, len(ct))
	this.Cbc.CryptBlocks(dst, ct)

	return __PKCS5Trimming(dst)
}

func __PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func __PKCS5Trimming(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}

func __DeriveKeyAndIv(passphrase string, salt string) ([]byte, []byte) {
	salted := ""
	dI := ""

	for len(salted) < 48 {
		md := md5.New()
		md.Write([]byte(dI + passphrase + salt))
		dM := md.Sum(nil)
		dI = string(dM[:16])
		salted = salted + dI
	}
	key := salted[0:32]
	iv := salted[32:48]

	return []byte(key), []byte(iv)
}

func RandUint(min uint, max uint) uint {
	if min > max {
		min, max = max, min
	}
	return (uint(mrd.Int63()) % (max - min + 1)) + min
}

/*
func wrecv(conn net.Conn) []byte {
	lenvSlc := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenvSlc); err != nil {
		return nil
	}
	reader := bytes.NewReader(lenvSlc)

	lenv := uint32(0)
	if err := binary.Read(reader, binary.LittleEndian, &lenv); err != nil {
		return nil
	}
	if lenv >= 102400 {
		panic("lenv beycond size")
	}
	slc := make([]byte, lenv)
	if _, err := io.ReadFull(conn, slc); err != nil {
		return nil
	}
	return slc
}

func wsend(conn net.Conn, data []byte) {
	outbuff := bytes.NewBuffer([]byte{})
	if err := binary.Write(outbuff, binary.LittleEndian, uint32(len(data))); err != nil {
		panic(err)
	}
	_, _ = conn.Write(outbuff.Bytes())
	_, _ = conn.Write(data)
}
func main() {

	listener, err := net.Listen("tcp", "0.0.0.0:8887")
	if err != nil {
		log.Printf("listen err %v\n", err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("accept err %v\n", err)
			break
		}
		handleConn(conn)
	}
}
func handleConn(conn net.Conn) {
	rsa := NewRsaCipher("pem/client_public.pem", "pem/server_private.pem")
	key0 := []byte(fmt.Sprintf("%v", RandUint(1000, 9999)))
	log.Println("send value:", string(key0))
	wsend(conn, rsa.Encrypt(key0))

	bts := wrecv(conn)
	keyGet := rsa.Decrypt(bts)
	if string(key0) != string(keyGet) {
		log.Printf("hand shake fail, send:%v recv: %v", key0, keyGet)
		conn.Close()
		return
	}
	log.Printf("handshake succ, key0:%v,keyget:%v", key0, keyGet)

	//process aes encrypt
	aesAlg := NewAesAlg(key0)
	for {
		bts = wrecv(conn)
		if bts == nil {
			conn.Close()
			break
		}
		decAry := aesAlg.Decrypt(bts)
		encAry := aesAlg.Encrypt(decAry)
		wsend(conn, encAry)
	}
	conn.Close()
}
*/

//all alg
type ICipher interface {
	Encrypt(src []byte) []byte
	Decrypt(src []byte) []byte
}

// AES
func NewAesCipher(key []byte) *AesCipher {
	var key32 [32]byte
	copy(key32[:], key)
	block, _ := aes.NewCipher(key32[:])
	return &AesCipher{key32: key32, block: block}
}

type AesCipher struct {
	key32 [32]byte
	block cipher.Block
}

func (this *AesCipher) Encrypt(src []byte) []byte {
	if len(src) == 0 {
		panic("nil src")
	}
	blockSize := this.block.BlockSize()
	src = __PKCS5Padding(src, blockSize)
	blockMode := cipher.NewCBCEncrypter(this.block, this.key32[:blockSize])
	dst := make([]byte, len(src))
	blockMode.CryptBlocks(dst, src)
	return dst
}

func (this *AesCipher) Decrypt(src []byte) []byte {
	if len(src) == 0 {
		panic("nil src")
	}
	blockSize := this.block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(this.block, this.key32[:blockSize])
	dst := make([]byte, len(src))
	blockMode.CryptBlocks(dst, src)
	dst = __PKCS5Trimming(dst)
	return dst
}

/*
// Padding
func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}
*/
