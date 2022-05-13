package netutils

import (
	"crypto"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/asmexie/gopub/cipher2"
	"github.com/asmexie/gopub/common"
)

// FieldTagCallback ...
type FieldTagCallback func(tag string, fdType *reflect.StructField, fdValue *reflect.Value) error

// WebParams ...
type WebParams struct {
	TimeStamp string `webvar:"timestamp"`
	Sig       string `webvar:"sign"`
	NonceStr  string `webvar:"nonce_str"`
	SignType  string `webvar:"sign_type"`
}

// NewWebParams ...
func NewWebParams(signType string) WebParams {
	return WebParams{}
}

func ssContains(a []string, x string) bool {
	x = strings.ToLower(x)
	for _, n := range a {
		if strings.ToLower(n) == x {
			return true
		}
	}
	return false
}

func foreachStructTags(data interface{}, tagName string, ignoreEmpty bool, cb FieldTagCallback) error {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}
	if dataType.Kind() != reflect.Struct {
		return errors.New("only support struct type")
	}

	for i := 0; i < dataType.NumField(); i++ {
		fdType := dataType.Field(i)
		tag := fdType.Tag.Get(tagName)
		if tag == "-" {
			continue
		}

		if ignoreEmpty && tag == "" {
			continue
		}
		fdValue := dataValue.Field(i)
		err := cb(tag, &fdType, &fdValue)
		if err != nil {
			return err
		}

	}
	return nil
}

// BuildValues ...
func BuildValues(webParams interface{}) (URLValues, error) {
	m := common.Map{}
	err := foreachStructTags(webParams, "webvar", true,
		func(tag string, fdType *reflect.StructField, fdValue *reflect.Value) error {
			m[tag] = fdValue.Interface()
			return nil
		})
	if err != nil {
		return URLValues{}, common.ERR(err)
	}
	values := MapToURLValues(m)
	return values, nil
}

// BuildSignedValues ...
func BuildSignedValues(webParams interface{}, signType, signKey string) (URLValues, error) {
	m := common.Map{}
	var signFD *reflect.Value
	err := foreachStructTags(webParams, "webvar", true,
		func(tag string, fdType *reflect.StructField, fdValue *reflect.Value) error {
			switch tag {
			case KNameNonceStr:
				nonceStr := common.RandStringRunes(16)
				fdValue.SetString(nonceStr)
			case KNameSign:
				signFD = fdValue
			case KNameSignType:
				fdValue.SetString(signType)
			}
			m[tag] = fdValue.Interface()
			return nil
		})
	if err != nil {
		return nil, err
	}
	values := MapToURLValues(m)
	values.Set(KNameNonceStr, common.RandStringRunes(16))
	sign, err := BuildValuesSign(values, signType, signKey)
	if err != nil {
		return nil, err
	}
	values.Set(KNameSign, sign)
	if signFD != nil {
		signFD.SetString(sign)
	}
	return values, nil
}

// VerifyValuesSign ...
func VerifyValuesSign(values URLValues, signType, signKey string) bool {
	signData := buildValuesSignData(values)
	signValue := values.Get(KNameSign)
	if signData == "" {
		return false
	}
	if signType == SignTypeMd5 {
		sign, err := signWithKey([]byte(signData), signType, signKey)
		if err != nil {
			common.LogERR(err)
			return false
		}
		return sign == signValue
	}

	sig, err := base64.StdEncoding.DecodeString(signValue)
	if err != nil {
		common.LogERR(err)
		return false
	}

	var hash crypto.Hash
	if signType == SignTypeRSA1 {
		hash = crypto.SHA1
	} else {
		hash = crypto.SHA256
	}
	ok, err := cipher2.VerifyPKCS1v15([]byte(signData), sig, []byte(signKey), hash)
	if err != nil {
		common.LogERR(err)
		return false
	}
	return ok
}

func md5Hash(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

func verifyWithKey(data []byte, sign string, hashType, signKey string) bool {
	switch hashType {
	case SignTypeMd5:
		data = append(data, []byte("&key="+signKey)...)
		return fmt.Sprintf("%x", md5Hash([]byte(data))) == sign
	case SignTypeRSA1, SignTypeRSA256:
		sig, err := base64.StdEncoding.DecodeString(sign)
		if err != nil {
			common.LogERR(err)
			return false
		}
		rsakey, err := base64.StdEncoding.DecodeString(signKey)
		if err != nil {
			common.LogERR(err)
			return false
		}
		var hash crypto.Hash
		if hashType == SignTypeRSA1 {
			hash = crypto.SHA1
		} else {
			hash = crypto.SHA256
		}
		ok, err := cipher2.VerifyPKCS1v15(data, sig, rsakey, hash)
		if err != nil {
			common.LogERR(err)
			return false
		}
		return ok
	default:
		return false
	}
}

func signWithKey(data []byte, signType, signKey string) (string, error) {
	switch signType {
	case SignTypeMd5:
		data = append(data, []byte("&key="+signKey)...)
		return fmt.Sprintf("%x", md5Hash([]byte(data))), nil
	case SignTypeRSA1, SignTypeRSA256:

		var hash crypto.Hash
		if signType == SignTypeRSA1 {
			hash = crypto.SHA1
		} else {
			hash = crypto.SHA256
		}
		sign, err := cipher2.SignPKCS1v15(data, []byte(signKey), hash)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(sign), nil
	default:
		return "", fmt.Errorf("not support sign type:%v", signType)
	}
}

func buildValuesSignData(values URLValues) string {
	keys := []string{}

	for key := range values {
		if key != KNameSign {
			keys = append(keys, key)
		}
	}

	sort.StringSlice(keys).Sort()
	data := ""
	for i, key := range keys {
		if i != 0 {
			data += "&"
		}
		data += (key + "=" + values.Get(key))
	}
	return data
}

// BuildValuesSign ...
func BuildValuesSign(values URLValues, hashType, signKey string) (string, error) {
	values.Set(KNameSignType, hashType)
	signData := buildValuesSignData(values)
	sign, err := signWithKey([]byte(signData), hashType, signKey)
	if err != nil {
		return "", err
	}

	return sign, nil
}

// ParseWebParams ...
func ParseWebParams(queryValues URLValues, webParams interface{}, discardFields ...string) (URLValues, error) {
	values := MakeURLValues(nil)
	return values, foreachStructTags(webParams, "webvar", true,
		func(tag string, fdType *reflect.StructField, fdValue *reflect.Value) error {
			if ssContains(discardFields, tag) {
				return nil
			}
			if !fdValue.IsValid() || !fdValue.CanSet() {
				return nil
			}
			v, ok := queryValues[tag]
			if !ok || len(v) == 0 {
				return nil
			}
			values.Set(tag, v[0])
			switch fdValue.Kind() {
			case reflect.Int, reflect.Int64:
				fdValue.SetInt(int64(values.IntValue(tag, 0)))
			case reflect.String:
				fdValue.SetString(values.Get(tag))
			case reflect.Float32, reflect.Float64:
				fdValue.SetFloat(values.FloatValue(tag, 0))
			default:
				return fmt.Errorf("tag %v is not support kind %v", tag, fdValue.Kind())
			}
			return nil
		})
}
