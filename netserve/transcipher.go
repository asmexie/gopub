package netserve

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/asmexie/gopub/cipher2"
	"github.com/asmexie/gopub/common"
)

type TransCipher interface {
	EncodeWrite(context *NetContext, buf *bufio.Writer, data []byte)
	DecodeRead(context *NetContext, buf *bufio.Reader) []byte
}

func checkArgsMinSize(args []string, size int) {
	if len(args) < size {
		panic(fmt.Errorf("arg  <<%s>> size less than %d", strings.Join(args, " "), size))
	}
}

func NewTransCipher(cipherCfg []string) TransCipher {
	checkArgsMinSize(cipherCfg, 1)
	switch cipherCfg[0] {
	case "nj11":
		checkArgsMinSize(cipherCfg, 3)
		return newEleCipher(cipherCfg[1], cipherCfg[2])
	case "sz12":
		checkArgsMinSize(cipherCfg, 2)
		key, err := base64.StdEncoding.DecodeString(cipherCfg[1])
		common.CheckError(err)
		return newszcipher(key)
	case "cccfg":
		checkArgsMinSize(cipherCfg, 2)
		return newCccfgCipher(cipherCfg[1])
	default:
		panic(fmt.Errorf("not support trans cipher type %s", cipherCfg[0]))
	}
}

type eleCipher struct {
	AesKey64 string
	AesIV64  string
	aeskeyb  []byte
	aesivb   []byte
	aesblock cipher.Block
}

func readLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		l, more, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		// Avoid the copy if the first call produced a full line.
		if line == nil && !more {
			return l, nil
		}
		line = append(line, l...)
		if !more {
			break
		}
	}
	return line, nil
}

func newEleCipher(aeskey64, aesiv64 string) *eleCipher {
	var err error
	ci := &eleCipher{}
	if aeskey64 != "" {
		ci.aeskeyb, err = base64.StdEncoding.DecodeString(aeskey64)
		common.CheckError(err)

		ci.aesblock, err = aes.NewCipher(ci.aeskeyb)
		common.CheckError(err)
	}

	if aesiv64 != "" {
		ci.aesivb, err = base64.StdEncoding.DecodeString(aesiv64)
		common.CheckError(err)
	}
	return ci
}

func (c *eleCipher) EncodeWrite(context *NetContext, buf *bufio.Writer, data []byte) {
	aes := cipher.NewCBCEncrypter(c.aesblock, c.aesivb)
	data, err := cipher2.ZeroPad([]byte(data), aes.BlockSize())
	common.CheckError(err)
	aes.CryptBlocks(data, data)

	buf.WriteString(base64.StdEncoding.EncodeToString(data) + "\r\n")
}

func (c *eleCipher) DecodeRead(context *NetContext, buf *bufio.Reader) []byte {
	data, err := readLine(buf)
	context.Verbose("recv data:" + string(data))
	common.CheckError(err)
	tmp, err := base64.StdEncoding.DecodeString(string(data))
	common.CheckError(err)
	aes := cipher.NewCBCDecrypter(c.aesblock, c.aesivb)
	aes.CryptBlocks(tmp, tmp)

	return []byte(tmp)
}

type szcipher struct {
	rsakeyb []byte
	rsaKey  *rsa.PrivateKey
	Seq     uint32
}

func newszcipher(rsakey []byte) *szcipher {
	var err error
	c := &szcipher{}
	c.rsakeyb = rsakey
	c.rsaKey, err = x509.ParsePKCS1PrivateKey(rsakey)
	c.Seq = rand.Uint32()
	common.CheckError(err)
	return c
}

func sizeof(t reflect.Type) int {
	switch t.Kind() {
	case reflect.Array:
		if s := sizeof(t.Elem()); s >= 0 {
			return s * t.Len()
		}

	case reflect.Struct:
		sum := 0
		for i, n := 0, t.NumField(); i < n; i++ {
			s := sizeof(t.Field(i).Type)
			if s < 0 {
				return -1
			}
			sum += s
		}
		return sum

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		return int(t.Size())
	}

	return -1
}

func readBytes(buf *bufio.Reader, n int) []byte {
	if n <= 0 {
		panic(fmt.Errorf("read n is %d", n))
	}
	var b []byte
	counter := 0
	for {
		tmp, err := buf.Peek(n)
		if len(tmp) > 0 {
			buf.Discard(len(tmp))
		}

		if err == nil && b == nil {
			return tmp
		}
		if err != nil && err != bufio.ErrBufferFull {
			panic(err)
		}
		if len(tmp) == 0 {
			counter += 1
			if counter > 10 {
				panic(fmt.Errorf("readBytes can not read any data, times %v", counter))
			}
			continue
		}
		b = append(b, tmp...)
		if err == nil {
			return b
		}
		n -= len(tmp)
	}
}

type TransPacketHdr struct {
	Checksum uint64
	Msgtype  uint32
	Version  uint32
	Seq      uint32
	Nonce    uint64
}

var testpackhdr TransPacketHdr
var packhdrsize = sizeof(reflect.TypeOf(testpackhdr))

func (c *szcipher) ReadAllData(rd *bufio.Reader) []byte {
	tmp := readBytes(rd, 4)

	length := int(binary.LittleEndian.Uint32(tmp))
	if length <= 0 {
		panic(fmt.Errorf("packet format erro"))
	}

	data := readBytes(rd, int(length))

	if len(data) != int(length) {
		panic(fmt.Errorf("packet format erro"))
	}
	return data
}

func (c *szcipher) GetSynAesIv(hdr TransPacketHdr, aesKeyB []byte) (iv []byte) {

	if hdr.Version == 1 {
		iv = make([]byte, 16)
		binary.LittleEndian.PutUint64(iv, hdr.Nonce)
		binary.LittleEndian.PutUint64(iv[8:], hdr.Nonce)
	} else if hdr.Version == 2 {
		h := md5.New()
		h.Write(aesKeyB)
		binary.Write(h, binary.LittleEndian, hdr.Nonce)
		binary.Write(h, binary.LittleEndian, hdr.Seq)

		iv = h.Sum(nil)

	} else {
		panic(fmt.Errorf("not support transfer version %v", hdr.Version))
	}
	return
}

func (c *szcipher) DecryptSyncData(context *NetContext, hdr TransPacketHdr, data []byte) (plain, aeskeyb, aesivb []byte) {
	k := (c.rsaKey.N.BitLen() + 7) / 8
	if len(data) < k {
		panic(fmt.Errorf("data lenth %d is valid", len(data)))
	}
	tmp := data[:k]
	context.Verbosef("rsa decrpyting data % x", tmp)

	de, err := rsa.DecryptPKCS1v15(nil, c.rsaKey, data[:k])

	if err != nil {
		context.Verbosef("decode rsa data failed % x", data)
		panic(err)
	}
	common.CheckError(err)
	context.Verbosef("rsa decrpyted data % x", de)
	aeskeyb = de[:16]
	aesivb = c.GetSynAesIv(hdr, aeskeyb)
	plain = de[16:]
	if len(data) == k {
		return
	}

	aesblock, err := aes.NewCipher(aeskeyb)
	common.CheckError(err)

	aes := cipher.NewCBCDecrypter(aesblock, aesivb)

	context.Verbosef("aes decrpyting \nkey % x \niv % x \ndata % x",
		aeskeyb,
		aesivb,
		data[k:])

	aes.CryptBlocks(data[k:], data[k:])
	context.Verbosef("aes decrpyted data % x", data[k:])
	if aesdata, err := cipher2.Pkcs7Unpad(data[k:], aes.BlockSize()); err != nil {
		common.CheckError(err)
	} else {
		plain = append(plain, aesdata...)
	}
	return
}

func RoundUp(size, bound int) int {
	return ((size + bound - 1) / bound) * bound
}

func (c *szcipher) EncryptAckData(context *NetContext, data []byte) (rs []byte) {
	aesblock, err := aes.NewCipher(context.aeskey)
	common.CheckError(err)
	iv := append([]byte{}, context.sendiv...)
	context.Verbosef("encrypt data, key % x, \n, iv % x,\n, data % x",
		context.aeskey, iv, data)
	aes := cipher.NewCBCEncrypter(aesblock, iv)

	if len(data) == 0 {
		panic(fmt.Errorf("encrypt data size is zero"))
	}
	tmp := make([]byte, len(data))
	copy(tmp, data)
	data, err = cipher2.Pkcs7Pad(tmp, aes.BlockSize())
	// context.Verbosef("aes encrypting \n key % x\n iv % x\n data % x",
	// 	context.aeskey, iv, data)
	common.CheckError(err)
	rs = make([]byte, len(data))
	aes.CryptBlocks(rs, data)
	//	context.Verbosef("aes encryypted data % x", rs)
	return
}

func (c *szcipher) WriteAckData(context *NetContext, buf *bufio.Writer, data []byte) {
	var hdr TransPacketHdr
	context.seq = atomic.AddUint32(&c.Seq, 1)

	context.BuildAckHdr(&hdr)

	encodedData := c.EncryptAckData(context, data)

	var newbuf bytes.Buffer
	var size uint32
	binary.Write(&newbuf, binary.LittleEndian, size)
	binary.Write(&newbuf, binary.LittleEndian, hdr)
	binary.Write(&newbuf, binary.LittleEndian, context.ack+1)
	if context.state == 2 {
		sig, err := cipher2.SignPKCS1v15WithKey(encodedData, c.rsaKey, crypto.MD5)
		common.CheckError(err)
		binary.Write(&newbuf, binary.LittleEndian, sig)
	}

	newbuf.Write(encodedData)

	newdata := newbuf.Bytes()

	if !context.stream {
		size = uint32(len(newdata) - 4)
	} else {
		if context.state == 2 {
			panic(errors.New("not supoort stream mode"))
		}
		if context.size < len(data) {
			panic(fmt.Errorf("not valid stream size %v", context.size))
		}
		size = uint32(context.CalcStreamSize(len(context.aeskey)))
		const acksize = 4
		size += uint32(packhdrsize + acksize)
	}

	binary.LittleEndian.PutUint32(newdata[:4], size)

	checkSum := c.CalcCheckSum(newdata[4:])
	binary.LittleEndian.PutUint64(newdata[4:12], checkSum)
	// context.Verbosef("writing ack data size %d hdr %+v ack %d ",
	// 	size, hdr, context.ack)

	//context.Verbosef("writing ack data % x", newdata)
	buf.Write(newdata)
	context.state++
}

func (c *szcipher) EncodeWrite(context *NetContext, buf *bufio.Writer, data []byte) {
	//context.Verbosef("EncodeWrite:% x", data)
	if context.state == 2 || context.state == 10 {
		c.WriteAckData(context, buf, data)
	} else {
		//logger.Debugf("write stream size %d data % x", len(data), data)
		context.UpdateIv()
		buf.Write(c.EncryptAckData(context, data))
	}
}

func (c *szcipher) CalcCheckSum(data []byte) uint64 {
	var checkSum uint64
	binary.LittleEndian.PutUint64(data, checkSum)
	h := md5.New()
	h.Write(data)
	ck := h.Sum(nil)
	return binary.LittleEndian.Uint64(ck[4:12])
}

func (c *szcipher) DecodeData(context *NetContext, data []byte) (rs []byte) {
	var hdr TransPacketHdr
	size := packhdrsize

	err := binary.Read(bytes.NewBuffer(data[:size]), binary.LittleEndian, &hdr)
	common.CheckError(err)

	checkSum := c.CalcCheckSum(data)
	if checkSum != hdr.Checksum {
		panic(fmt.Errorf("packet checksum error hdr %+v s %x and c %x and data % x ", hdr,
			checkSum, hdr.Checksum, data))
	}
	if !context.checkSetAck(hdr.Seq) {
		context.Verbosef("receive repeat seq %d data", hdr.Seq)
		return
	}

	context.ack = hdr.Seq
	context.recvsig = hdr.Checksum

	var aeskeyb []byte
	rs, aeskeyb, context.recviv = c.DecryptSyncData(context, hdr, data[size:])

	context.aeskey = append([]byte{}, aeskeyb...)
	context.updateiv = true
	if hdr.Msgtype == 1 {
		context.state = 2
	} else {
		context.state = 10
	}
	return
}

func (c *szcipher) DecodeRead(context *NetContext, rd *bufio.Reader) (rs []byte) {
	data := c.ReadAllData(rd)
	context.Verbosef("readed data:% x", data)

	rs = c.DecodeData(context, data)
	return
}

type emptycipher struct {
}

func (c *emptycipher) EncodeWrite(context *NetContext, buf *bufio.Writer, data []byte) {
	buf.Write(data)
}

func (c *emptycipher) DecodeRead(context *NetContext, buf *bufio.Reader) []byte {
	var data []byte
	for {
		tmp := make([]byte, 1024)
		n, err := buf.Read(tmp)
		if err == io.EOF || err == nil {
			if n > 0 {
				if data == nil {
					data = tmp[:n]
				} else {
					data = append(data, tmp[:n]...)
				}

			}
			if err == io.EOF {
				return data
			}
		} else if err != nil {
			panic(err)
		}
	}
}

type cccfgcipher struct {
	AesKey64 string
	aeskeyb  []byte
	aesblock cipher.Block
}

func newCccfgCipher(key64 string) *cccfgcipher {
	var err error
	if key64 != "" {
		ci := &cccfgcipher{}
		ci.AesKey64 = key64
		ci.aeskeyb, err = base64.StdEncoding.DecodeString(key64)
		common.CheckError(err)

		ci.aesblock, err = aes.NewCipher(ci.aeskeyb)
		common.CheckError(err)
		return ci

	} else {
		return nil
	}
}

func (c *cccfgcipher) ReadAll(rd *bufio.Reader) []byte {
	data := make([]byte, 4096)
	n, err := rd.Read(data)

	if err != nil && err != io.EOF {
		panic(err)
	}
	if n == 0 {
		panic(errors.New("read empty data"))
	}
	return data[:n]
}

func (c *cccfgcipher) EncodeWrite(context *NetContext, buf *bufio.Writer, data []byte) {
	aes := cipher2.NewECBEncrypter(c.aesblock)
	data, err := cipher2.ZeroPad([]byte(data), aes.BlockSize())
	common.CheckError(err)
	aes.CryptBlocks(data, data)
	buf.WriteString(base64.StdEncoding.EncodeToString(data))
}

func (c *cccfgcipher) DecodeRead(context *NetContext, buf *bufio.Reader) []byte {
	base64data := c.ReadAll(buf)
	if len(base64data) == 0 {
		panic(errors.New("read empty data"))
	}
	data, err := base64.StdEncoding.DecodeString(string(base64data))
	common.CheckError(err)
	aes := cipher2.NewECBDecrypter(c.aesblock)
	aes.CryptBlocks(data, data)
	return data
}
