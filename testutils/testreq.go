package testutils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/gopub/netutils"
	"github.com/muroachanf/go-logger/logger"
)

// BeforeTestReqRunProc ...
type BeforeTestReqRunProc func(wr *TestWebReq, req *http.Request)

// AfterTestReqRunProc ...
type AfterTestReqRunProc func(wr *TestWebReq, w *httptest.ResponseRecorder)

// TestWebReq ...
type TestWebReq struct {
	reqURL      string
	handler     http.Handler
	req         *http.Request
	w           *httptest.ResponseRecorder
	cookies     []*http.Cookie
	method      string
	values      netutils.URLValues
	result      []byte
	haveRead    bool
	body        io.ReadCloser
	contentType string
	beforeRuns  []BeforeTestReqRunProc
	afterRuns   []AfterTestReqRunProc
}

// NewTestWebReq ...
func NewTestWebReq(h http.Handler, method string, reqURL string) *TestWebReq {
	return &TestWebReq{
		reqURL:  reqURL,
		handler: h,
		w:       httptest.NewRecorder(),
		method:  method,
	}
}

func addCookies(req *http.Request, cookies []*http.Cookie) {
	for _, c := range cookies {
		req.AddCookie(c)
	}
}

// Req ...
func (wr *TestWebReq) Req() *http.Request {
	return wr.req
}

// SetCookies ...
func (wr *TestWebReq) SetCookies(cookies []*http.Cookie) *TestWebReq {
	wr.cookies = cookies
	return wr
}

// SetBody ...
func (wr *TestWebReq) SetBody(body io.Reader) *TestWebReq {
	bd, ok := body.(io.ReadCloser)
	if ok {
		wr.body = bd
	} else {
		wr.body = ioutil.NopCloser(body)
	}
	return wr
}

// SetContentType ...
func (wr *TestWebReq) SetContentType(contentType string) *TestWebReq {
	//wr.req.Header.Set("Content-Type", contentType)
	wr.contentType = contentType
	return wr
}

// Recorder ...
func (wr *TestWebReq) Recorder() *httptest.ResponseRecorder {
	return wr.w
}

func (wr *TestWebReq) checkReadResult() {
	if !wr.haveRead && wr.w.Result().ContentLength != 0 {
		s, err := ioutil.ReadAll(wr.w.Body)
		common.CheckError(err)
		wr.result = s
		wr.haveRead = true
	}
}

// BeforeRun ...
func (wr *TestWebReq) BeforeRun(runBefore BeforeTestReqRunProc) *TestWebReq {
	wr.beforeRuns = append(wr.beforeRuns, runBefore)
	return wr
}

// AfterRun ...
func (wr *TestWebReq) AfterRun(runAfter AfterTestReqRunProc) *TestWebReq {
	wr.afterRuns = append(wr.afterRuns, runAfter)
	return wr
}

// SetValues ...
func (wr *TestWebReq) SetValues(values netutils.URLValues) *TestWebReq {
	wr.values = values
	return wr
}

// SetWebParams ...
func (wr *TestWebReq) SetWebParams(wp interface{}) *TestWebReq {
	values, err := netutils.BuildValues(wp)
	common.CheckError(err)
	wr.values = values
	return wr
}

// Run ...
func (wr *TestWebReq) Run() *TestWebReq {
	var err error
	reqURL := wr.reqURL
	if wr.method == http.MethodGet && wr.values != nil {
		if -1 == strings.Index(reqURL, "?") {
			reqURL += "?"
		} else {
			reqURL += "&"
		}
		reqURL += wr.values.Encode()

	}
	body := wr.body
	if wr.method == http.MethodPost {
		if wr.values != nil {
			body = ioutil.NopCloser(bytes.NewBuffer([]byte(wr.values.Encode())))
		}
	}
	wr.req, err = http.NewRequest(wr.method, reqURL, body)
	common.CheckError(err)

	if wr.contentType != "" {
		wr.req.Header.Set("Content-Type", wr.contentType)
	}

	addCookies(wr.req, wr.cookies)
	for _, bf := range wr.beforeRuns {
		bf(wr, wr.req)
	}
	wr.handler.ServeHTTP(wr.w, wr.req)
	wr.checkReadResult()

	for _, af := range wr.afterRuns {
		af(wr, wr.w)
	}

	return wr
}

// CheckResult ...
func (wr *TestWebReq) CheckResult() *TestWebReq {
	if wr.w.Code != 200 {
		common.CheckError(fmt.Errorf("req failed with error:%v", wr.w.Code))
	}
	return wr
}

// ResultASString ...
func (wr *TestWebReq) ResultASString() string {
	return string(wr.result)
}

// ResultAsMap ...
func (wr *TestWebReq) ResultAsMap() common.Map {
	defer func() {
		if e := recover(); e != nil {
			logger.Debugf("read map result failed:%v", string(wr.result))
			panic(e)
		}
	}()
	return common.ReadMap(bytes.NewBuffer(wr.result), false)
}

// PrintResult ...
func (wr *TestWebReq) PrintResult(w io.Writer) *TestWebReq {
	io.WriteString(w, string(wr.result))
	return wr
}
