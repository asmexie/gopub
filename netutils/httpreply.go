package netutils

import (
	"net/http"
)

const (
	cDefaultMsgKey    = "msg"
	cDefaultStatusKey = "status"
)

//HTTPReplyJSON ...
type HTTPReplyJSON struct {
	httpMsgInfo map[int]string
	Reply       map[string]interface{}
	msgKey      string
	statusKey   string
}

//SetStatus ...
func (reply *HTTPReplyJSON) SetStatus(code int) {
	reply.Reply[reply.statusKey] = code
	reply.Reply[reply.msgKey] = reply.httpMsgInfo[code]
}

// GetStatus ...
func (reply *HTTPReplyJSON) GetStatus() int {
	if v, ok := reply.Reply[reply.statusKey]; ok {
		return v.(int)
	}
	return 0
}

//SetStatusAndMsg ...
func (reply *HTTPReplyJSON) SetStatusAndMsg(code int, msg string) {
	reply.Reply[reply.statusKey] = code
	reply.Reply[reply.msgKey] = msg
}

//SetStatusAndData ...
func (reply *HTTPReplyJSON) SetStatusAndData(code int, data interface{}) {
	reply.Reply[reply.statusKey] = code
	reply.Reply[reply.msgKey] = reply.httpMsgInfo[code]
	reply.Reply["data"] = data
}

//SetStatusAndMsgAndData ...
func (reply *HTTPReplyJSON) SetStatusAndMsgAndData(code int, msg string, data interface{}) {
	reply.Reply[reply.statusKey] = code
	reply.Reply[reply.msgKey] = msg
	reply.Reply["data"] = data
}

// WriteReply ...
func (reply *HTTPReplyJSON) WriteReply(w http.ResponseWriter) {
	WriteJSON(w, reply.Reply)
}

// NewHTTPReplyJSONEx ...
func NewHTTPReplyJSONEx(msgMap map[int]string, defaultErr int, msgKey, statusKey string) *HTTPReplyJSON {
	return &HTTPReplyJSON{
		httpMsgInfo: msgMap,
		Reply: map[string]interface{}{
			statusKey: defaultErr,
			msgKey:    msgMap[defaultErr],
		},
		msgKey:    msgKey,
		statusKey: statusKey,
	}
}

// NewHTTPReplyJSON ...
func NewHTTPReplyJSON(msgMap map[int]string, defaultErr int) *HTTPReplyJSON {
	return NewHTTPReplyJSONEx(msgMap, defaultErr, cDefaultMsgKey, cDefaultStatusKey)
}
