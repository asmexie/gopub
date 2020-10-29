package cipher2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/binary"
	"errors"
)

// ErrDataSizeIsZero ...
var ErrDataSizeIsZero = errors.New("data size is zero")

// AesDecrypt ...
func AesDecrypt(aesKey, iv, data []byte) ([]byte, error) {
	aesKey = append([]byte{}, aesKey...)
	iv = append([]byte{}, iv...)
	if len(data) == 0 {
		panic(ErrDataSizeIsZero)
	}
	aesblock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	decrypter := cipher.NewCBCDecrypter(aesblock, iv)
	rs := make([]byte, len(data))
	decrypter.CryptBlocks(rs, data)
	return Pkcs7Unpad(rs, decrypter.BlockSize())
}

// AesEncrypt ...
func AesEncrypt(aesKey, iv, data []byte) ([]byte, error) {
	aesKey = append([]byte{}, aesKey...)
	iv = append([]byte{}, iv...)
	aesblock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	encrypter := cipher.NewCBCEncrypter(aesblock, iv)
	data, err = Pkcs7Pad(data, encrypter.BlockSize())
	if err != nil {
		return nil, err
	}
	rs := make([]byte, len(data))
	encrypter.CryptBlocks(rs, data)
	return rs, nil
}

// Md5HashBytes ...
func Md5HashBytes(d []byte) []byte {
	hash := md5.New()
	hash.Write(d)
	return hash.Sum(nil)
}

// Md5HashStr ...
func Md5HashStr(s string) []byte {
	return Md5HashBytes([]byte(s))
}

// Md5HashObjsLi ...
func Md5HashObjsLi(byteOrder binary.ByteOrder, vlist ...interface{}) []byte {
	h := md5.New()
	for _, v := range vlist {
		binary.Write(h, byteOrder, v)
	}
	return h.Sum(nil)
}

// SimpleCrypt ...
func SimpleCrypt(data, key []byte, encrypt bool) []byte {
	delta := uint32(0x3f)
	var sum uint32
	var v0, v1 uint32
	for i := 0; i < len(data); i++ {
		k1 := uint32(key[i%len(key)])
		k2 := uint32(key[(i+1)%len(key)])

		for j := 0; j < 8; j++ {
			sum += delta
			v0 += ((v1 << 4) + k1) ^ (v1 + sum) ^ ((v1 >> 5) + k2)
			v1 += ((v0 << 4) + k1) ^ (v0 + sum) ^ ((v0 >> 5) + k2)
		}
		if encrypt {
			data[i] ^= byte(v1)
			sum += uint32(data[i])
		} else {
			sum += uint32(data[i])
			data[i] ^= byte(v1)
		}

	}

	return data
}
