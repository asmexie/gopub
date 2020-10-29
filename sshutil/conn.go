package sshutil

import (
	"net"
	"time"
)

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type TimeoutConn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func (c *TimeoutConn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	n, err := c.Conn.Read(b)
	if err != nil {
		// logger.Debugf("read %d,%d local %v remote %v failed %v", len(b), n, c.LocalAddr().String(), c.RemoteAddr().String(), err)
	}
	return n, err
}

func (c *TimeoutConn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	n, err := c.Conn.Write(b)
	if err != nil {
		// logger.Debugf("write size %d,%d local %v remote %v failed %v", len(b), n, c.LocalAddr().String(), c.RemoteAddr().String(), err)
	}
	return n, err
}
