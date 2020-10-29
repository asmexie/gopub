package netutils

import (
	"encoding/json"
	"net"
	"net/http"
)

// WriteJSON ...
func WriteJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

// GetQueryValues ...
func GetQueryValues(r *http.Request) URLValues {
	if r.Method == "GET" {
		return URLValues(r.URL.Query())
	}
	if r.Form == nil {
		r.ParseMultipartForm(32 << 20)
	}
	return URLValues(r.Form)
}

const (
	cXForwardedFor = "X-Forwarded-For"
	cXRealIP       = "X-Real-IP"
)

// RemoteIP ...
func RemoteIP(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(cXRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(cXForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}
