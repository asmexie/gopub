package netserve

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/asmexie/gopub/common"
	"github.com/muroachanf/go-logger/logger"
)

// PDecoder ...
type PDecoder interface {
	Decode(buf []byte) (api int, data []byte, err error)
}

// BaseDecoder ...
type BaseDecoder struct {
	nsc NetServeConfig
	hd  APIHandler
}

// ConvertSApiToCode ...
func (d BaseDecoder) ConvertSApiToCode(apis string) int {
	return d.hd.ConvertSApiToCode(apis)
}

// ConvertApiToCode ...
func (d BaseDecoder) ConvertApiToCode(api int) int {
	return d.hd.ConvertAPIToCode(api)
}

// QueryAppSecretKey ...
func (d BaseDecoder) QueryAppSecretKey(app string) string {
	return d.hd.QueryAppSecretKey(app)
}

type elepdecoder struct {
	BaseDecoder
}

func newDecoder(nsc NetServeConfig, hd APIHandler) PDecoder {
	switch nsc.CodeType {
	case "nj11":
		return &elepdecoder{BaseDecoder: BaseDecoder{nsc: nsc, hd: hd}}
	case "sz12":
		return &szpdecoder{BaseDecoder: BaseDecoder{nsc: nsc, hd: hd}}
	case "mt":
		return &mtpdecoder{BaseDecoder: BaseDecoder{nsc: nsc, hd: hd}}
	default:
		panic(fmt.Errorf("not support trans cipher type %s", nsc.CodeType))
	}
}

func (*elepdecoder) VerifyValues(v *url.Values) (data string, result bool) {

	data = v.Get("data")
	if data == "" {
		data = v.Get("Data")
	}
	result = data != ""
	return
}

func (d *elepdecoder) Decode(buf []byte) (api int, data []byte, err error) {
	defer func() {
		if err != nil {
			common.LogError(err)
		}
	}()
	valueText := string(bytes.Trim(buf, "\x00"))

	v, err := url.ParseQuery(valueText)
	if err != nil {
		logger.Debugf("ParseQuery msg failed:%s", valueText)
		return
	}

	dataS, ret := d.VerifyValues(&v)
	if !ret {
		logger.Debugf("VerifyValues msg failed:%s", valueText)
		return
	}

	//logger.Debug("recv data:" + dataS)
	p, err := base64.StdEncoding.DecodeString(dataS)
	if err != nil {
		logger.Error("parse failed with data:" + dataS)
		return
	}
	api = d.ConvertSApiToCode(v.Get("type"))
	data = p
	return
}

type szApiData struct {
	Api  string
	Data json.RawMessage
}

type szpdecoder struct {
	BaseDecoder
}

func (d *szpdecoder) Decode(buf []byte) (api int, data []byte, err error) {

	var apiData szApiData

	if buf == nil || len(buf) == 0 {
		logger.Debug("decoding  empty data")
		return
	}
	s := bytes.Trim(buf, "\x00")
	if len(s) == 0 {
		logger.Debug("decoding  empty data")
	}
	p := strings.Replace(string(s), "\n", "", -1)
	//p = strings.Replace(p, "\\", "\\\\", -1)
	if d.nsc.LogVerbose {
		logger.Debug("decoding data " + p)
	}

	err = json.Unmarshal([]byte(p), &apiData)
	if err != nil {
		logger.Debugf("decoding failed data % x", []byte(p))
		logger.Debug("decoding failed data " + p)
		common.CheckError(err)
	}

	api = d.ConvertSApiToCode(apiData.Api)
	data = apiData.Data

	return
}

type WebApiData struct {
	Api   string
	App   string
	Nonce uint64
	Data  json.RawMessage
	Sig   string
}

func (d *webdecoder) CalcSig(apidata WebApiData) string {
	secrectKey := d.QueryAppSecretKey(apidata.App)
	p := fmt.Sprintf("%v&%v&%v&%v&%v", apidata.Api, apidata.App, apidata.Nonce, string(apidata.Data), secrectKey)
	//logger.Debug("calc sig str:" + p)
	h := md5.New()
	h.Write([]byte(p))
	ck := h.Sum(nil)
	return fmt.Sprintf("%x", ck)
}

type webdecoder struct {
	BaseDecoder
}

func (d *webdecoder) CheckSig(apidata WebApiData) {
	sig := d.CalcSig(apidata)
	if strings.ToLower(sig) != strings.ToLower(apidata.Sig) {
		panic(fmt.Sprintf("sig %v is error, mine %v", apidata.Sig, sig))
	}
}

func (d *webdecoder) Decode(buf []byte) (api int, data []byte, err error) {
	s := string(bytes.Trim(buf, "\x00"))
	logger.Debugf("recv web msg %s", s)
	tmp, err := base64.StdEncoding.DecodeString(s)
	common.CheckError(err)
	logger.Debugf("decoding web msg %s", string(tmp))
	var apidata WebApiData
	err = json.Unmarshal(tmp, &apidata)
	common.CheckError(err)
	d.CheckSig(apidata)
	api = d.ConvertSApiToCode(apidata.Api)
	data = apidata.Data
	return
}

type mtpdecoder struct {
	BaseDecoder
}

func (d *mtpdecoder) Decode(buf []byte) (api int, data []byte, err error) {
	//logger.Debugf("mt decoding data:% x", buf)
	api = d.ConvertApiToCode(int(binary.LittleEndian.Uint16(buf)))
	//logger.Debugf("mt got api:%d", api)
	data = buf[2:]
	return
}
