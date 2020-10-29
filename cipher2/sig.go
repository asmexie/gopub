package cipher2

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
)

// ParsePKCS1PrivateKey ...
// func ParsePKCS1PrivateKey(data []byte) (key *rsa.PrivateKey, err error) {
// 	var block *pem.Block
// 	block, _ = pem.Decode(data)
// 	if block == nil {
// 		return nil, errors.New("private key error")
// 	}

// 	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return key, err
// }

// ParseBase64PrivateKey ...
func ParseBase64PrivateKey(data []byte) (key *rsa.PrivateKey, err error) {
	keyBytes, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	key, err = x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return key, err
}

// SignPKCS1v15WithKey ...
func SignPKCS1v15WithKey(src []byte, key *rsa.PrivateKey, hash crypto.Hash) ([]byte, error) {
	var h = hash.New()
	h.Write(src)
	var hashed = h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, key, hash, hashed)
}

// SignPKCS1v15 ...
func SignPKCS1v15(src, key []byte, hash crypto.Hash) ([]byte, error) {
	pri, err := ParseBase64PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return SignPKCS1v15WithKey(src, pri, hash)
}

// VerifyPKCS1v15WithKey ...
func VerifyPKCS1v15WithKey(src, sig []byte, key *rsa.PublicKey, hash crypto.Hash) error {
	var h = hash.New()
	h.Write(src)
	var hashed = h.Sum(nil)
	return rsa.VerifyPKCS1v15(key, hash, hashed, sig)
}

// VerifyPKCS1v15 ...
func VerifyPKCS1v15(src, sign, key []byte, hash crypto.Hash) (bool, error) {
	pubkey, err := ParseBase64PublicKey(key)
	if err != nil {
		return false, err
	}
	err = VerifyPKCS1v15WithKey(src, sign, pubkey, hash)
	return err == nil, err
}

// ParsePKCS1PublicKey ...
// func ParsePKCS1PublicKey(data []byte) (key *rsa.PublicKey, err error) {
// 	var block *pem.Block
// 	block, _ = pem.Decode(data)
// 	if block == nil {
// 		return nil, errors.New("private key error")
// 	}

// 	key, err = x509.ParsePKCS1PublicKey(block.Bytes)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return key, err
// }

// ParseBase64PublicKey ...
func ParseBase64PublicKey(data []byte) (key *rsa.PublicKey, err error) {
	keyBytes, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	key, err = x509.ParsePKCS1PublicKey(keyBytes)
	if err != nil {
		return nil, err
	}

	return key, err
}
