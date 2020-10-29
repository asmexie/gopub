package netserve

import (
	"bytes"
	"context"
	"net"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/muroachanf/gopub/common"
	"github.com/muroachanf/go-logger/logger"
)

type NetServe interface {
	Serve(ctx context.Context)
}

func newNetServe(serve *ServeGroup, nettype, ip string, port int) NetServe {
	if strings.Contains(nettype, "tcp") {
		s := &tcpserve{ServeGroup: serve}
		s.Listen(nettype, ip, port)
		return s
	} else {
		s := &udpserve{ServeGroup: serve}
		s.Listen(nettype, ip, port)
		return s
	}

}

type tcpserve struct {
	*ServeGroup
	listener net.Listener
}

func (s *tcpserve) Listen(nettype, ip string, port int) {
	laddr := ip + ":" + strconv.Itoa(port)
	logger.Infof("start listen %s on %s ctype %s dtype %s",
		nettype, laddr, s.nsc.Cipher[0], s.nsc.CodeType)
	l, err := net.Listen(nettype, laddr)
	common.CheckError(err)
	s.listener = l
}

func (s *tcpserve) newTcpConn(rwc net.Conn) (c *conn) {
	rt := time.Duration(s.nsc.ReadTimeOut) * time.Second
	wt := time.Duration(s.nsc.ReadTimeOut) * time.Second
	c = newConn(&tcpconn{c: rwc, sg: s.ServeGroup,
		readTimeOut: rt, writeTimeOut: wt},
		s.ServeGroup, true)
	c.readTimeOut = rt
	c.writeTimeOut = wt
	c.context.logVerbose = s.nsc.LogVerbose
	return c
}

func (s *tcpserve) Serve(ctx context.Context) {
	l := s.listener
	defer func() {
		if x := recover(); x != nil {
			common.LogError(x)
		}
		l.Close()
	}()
	logger.Debugf("start tcp serve at %d", s.nsc.Port)
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			common.LogError(err)
			//fmt.Println("Error accepting: ", err.Error())
			continue
		}

		// Handle connections in a new goroutine.
		c := s.newTcpConn(conn)
		go c.HandleRequest()
		if s.terminate {
			break
		}
	}
}

type udpserve struct {
	*ServeGroup
	conn *net.UDPConn
}

func (s *udpserve) newUdpConn(data []byte, addr *net.UDPAddr) (c *conn) {
	c = newConn(&udpconn{c: s.conn, sg: s.ServeGroup, addr: addr, data: bytes.NewBuffer(data)},
		s.ServeGroup, false)
	c.readTimeOut = time.Duration(s.nsc.ReadTimeOut) * time.Second
	c.context.logVerbose = s.nsc.LogVerbose
	return c
}

func (s *udpserve) Listen(nettype, ip string, port int) {
	tp := s.nsc
	laddr := ip + ":" + strconv.Itoa(port)

	addr, err := net.ResolveUDPAddr("udp", laddr)
	common.CheckError(err)
	logger.Infof("start listen %s on %s ctype %s dtype %s",
		nettype, laddr, tp.Cipher[0], tp.CodeType)

	l, err := net.ListenUDP(nettype, addr)

	common.CheckError(err)
	s.conn = l
}

func (s *udpserve) Serve(ctx context.Context) {
	conn := s.conn
	defer func() {
		if x := recover(); x != nil {
			common.LogError(x)
		}
		conn.Close()
	}()

	t := time.Duration(s.nsc.ReadTimeOut) * time.Second
	verbose := s.nsc.LogVerbose

	logger.Debugf("start udp serve at %d", s.nsc.Port)

	for {
		buf := make([]byte, 1024)
		if t != 0 {
			conn.SetDeadline(time.Now().Add(t))
		}

		n, addr, err := conn.ReadFromUDP(buf)

		if err != nil {
			if !strings.Contains(err.Error(), "timeout") {
				common.LogError(err)
			}
			continue
		}
		if n == 0 {
			logger.Error("read zero size udp packet")
			continue
		} else if verbose {
			logger.Debugf("readed udp data %d", n)
		}
		c := s.newUdpConn(buf[:n], addr)
		go c.HandleRequest()
		if s.terminate {
			break
		}
	}
}
