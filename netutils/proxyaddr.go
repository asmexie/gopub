package netutils

import (
	"fmt"
	"strings"
)

const (
	PXYProtocolHTTP   = "http"
	PXYProtocolHTTPS  = "https"
	PXYProtocolSocks5 = "socks5"
)

// ProxyAddr ...
type ProxyAddr struct {
	Protocol string
	Host     string
	Port     string
}

// ParseProxyAddr ...
func ParseProxyAddr(addr string) (proxyAddr ProxyAddr, err error) {
	if addr == "" {
		return
	}

	ss := strings.Split(addr, ":")

	if len(ss) == 1 {
		proxyAddr.Protocol = PXYProtocolHTTP
		proxyAddr.Port = "80"
		proxyAddr.Host = addr
		return

	}
	if len(ss) == 2 {
		if ss[1][0] == '/' {
			proxyAddr.Protocol = strings.ToLower(ss[0])
			proxyAddr.Host = ss[1][2:]
			switch proxyAddr.Protocol {
			case PXYProtocolHTTP:
				proxyAddr.Port = "80"
			case PXYProtocolHTTPS:
				proxyAddr.Port = "443"
			case PXYProtocolSocks5:
				proxyAddr.Port = "1080"
			default:
				proxyAddr.Port = "80"
			}
			return
		}
		proxyAddr.Protocol = PXYProtocolHTTP
		proxyAddr.Host = ss[0]
		proxyAddr.Port = ss[1]
		return
	}
	if len(ss) == 3 {
		proxyAddr.Protocol = ss[0]
		proxyAddr.Host = ss[1][2:]
		proxyAddr.Port = ss[2]
		return
	}
	err = fmt.Errorf("not suport proxyaddr:%v", addr)
	return
}
