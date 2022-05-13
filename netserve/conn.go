package netserve

import (
	"bufio"
	"bytes"

	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/go-logger/logger"
)

const noLimit int64 = (1 << 63) - 1

// SimpleNetConn ...
type SimpleNetConn interface {
	Read() []byte
	Write(p []byte) (n int, err error)
	PeerAddr() string
	// TransCipher() TransCipher
	// Decoder() PDecoder
}

// RawNetConn ...
type RawNetConn interface {
	io.ReadWriteCloser
	PeerAddr() string
}

type conn struct {
	c    RawNetConn
	buf  *bufio.ReadWriter
	lr   *io.LimitedReader
	w    io.Writer
	rsw  http.ResponseWriter
	werr error
	sg   *ServeGroup

	readTimeOut  time.Duration
	writeTimeOut time.Duration
	context      *NetContext
	handler      APIHandler
}

func newConn(netconn RawNetConn, sg *ServeGroup, isTCP bool) (c *conn) {
	c = new(conn)
	c.sg = sg
	c.c = netconn
	c.w = netconn
	c.handler = sg.hd

	var ok bool
	if c.rsw, ok = netconn.(http.ResponseWriter); !ok {
		c.rsw = nil
	}
	c.lr = io.LimitReader(netconn, noLimit).(*io.LimitedReader)
	br := newBufioReader(c.lr)
	bw := newBufioWriter(checkConnErrorWriter{c}, 4<<10)
	c.buf = bufio.NewReadWriter(br, bw)
	if isTCP {
		c.context = NewNetContext(netconn.PeerAddr())
	} else {
		c.context = GetUdpNetContext(netconn.PeerAddr())
	}
	return c
}

func (c *conn) PeerAddr() string {
	return c.c.PeerAddr()
}

// PeerAddrIP ...
func PeerAddrIP(addr string) (ip string) {
	ips := strings.Split(addr, ":")

	if len(ips) > 1 {
		ip = ips[0]
	} else {
		ip = addr
	}
	return
}

func (c *conn) BeginWriteStream(size int, packsize int) {
	c.context.stream = true
	c.context.size = size
	c.context.packsize = packsize
}

func (c *conn) finalFlush() {
	if c.buf != nil {
		c.buf.Flush()

		// Steal the bufio.Reader (~4KB worth of memory) and its associated
		// reader for a future connection.
		putBufioReader(c.buf.Reader)

		// Steal the bufio.Writer (~4KB worth of memory) and its associated
		// writer for a future connection.
		putBufioWriter(c.buf.Writer)

		c.buf = nil
	}
}

func (c *conn) Close() (err error) {
	c.finalFlush()
	if c.c != nil {
		err = c.c.Close()
		c.c = nil
	}
	return
}
func (c *conn) Write(data []byte) (int, error) {
	if c.rsw != nil {
		logger.Debug("writing status ok")
		c.rsw.WriteHeader(http.StatusOK)
	}
	if len(data) > 0 {
		//logger.Debugf("writing data % x", s)
		c.sg.cipher.EncodeWrite(c.context, c.buf.Writer, data)
		return len(data), nil
	}
	return 0, nil
}

func (c *conn) Read() []byte {
	return c.sg.cipher.DecodeRead(c.context, c.buf.Reader)
}

func (c *conn) HandleRequest() {
	defer func() {
		if x := recover(); x != nil {
			errmsg := x.(error).Error()
			if !strings.Contains(errmsg, "timeout") {
				common.LogError(x)
			}

		}
		c.Close()
	}()
	c.context.Verbosef("start read data from new conn")
	rawData := c.Read()
	if rawData == nil || len(rawData) == 0 {
		c.context.Verbosef("receive data is empty")
		return
	}
	api, data, err := c.sg.d.Decode(rawData)
	if err != nil {
		common.LogError(err)
		return
	}
	c.context.Verbosef("recv ip %v api %v data:% x\n", c.c.PeerAddr(), api, data)
	c.handler.HandleAPI(c, api, data)
}

type checkConnErrorWriter struct {
	c *conn
}

func (w checkConnErrorWriter) Write(p []byte) (n int, err error) {
	n, err = w.c.c.Write(p) // c.w == c.rwc, except after a hijack, when rwc is nil.
	if err != nil && w.c.werr == nil {
		w.c.werr = err
	}
	return
}

type tcpconn struct {
	c            net.Conn
	readTimeOut  time.Duration
	writeTimeOut time.Duration
	sg           *ServeGroup
}

func (c *tcpconn) Read(p []byte) (n int, err error) {
	if c.readTimeOut != 0 {
		c.c.SetReadDeadline(time.Now().Add(c.readTimeOut))
	}

	return c.c.Read(p)
}

func (c *tcpconn) Write(p []byte) (n int, err error) {
	if c.writeTimeOut != 0 {
		c.c.SetWriteDeadline(time.Now().Add(c.writeTimeOut))
	}
	return c.c.Write(p)
}

func (c *tcpconn) Close() error {
	return c.c.Close()
}

func (c *tcpconn) TransCipher() TransCipher {
	return c.sg.cipher
}

func (c *tcpconn) Decoder() PDecoder {
	return c.sg.d
}

func (c *tcpconn) PeerAddr() string {
	return c.c.RemoteAddr().String()
}

type udpconn struct {
	data *bytes.Buffer
	c    *net.UDPConn
	addr *net.UDPAddr
	sg   *ServeGroup
}

func (c *udpconn) Read(p []byte) (n int, err error) {
	return c.data.Read(p)
}

func (c *udpconn) Write(p []byte) (n int, err error) {
	if c.sg.nsc.Debug == 1 {
		logger.Debugf("udp writing data % x", p)
		logger.Debug("in udp debug state, sleep 2 second")
		time.Sleep(2 * time.Second)
	}
	return c.c.WriteToUDP(p, c.addr)
}

func (c *udpconn) Close() error {
	return nil
}

func (c *udpconn) TransCipher() TransCipher {
	return c.sg.cipher
}

func (c *udpconn) Decoder() PDecoder {

	return c.sg.d
}

func (c *udpconn) PeerAddr() string {
	return c.addr.String()
}

type webconn struct {
	data *bytes.Buffer
	w    http.ResponseWriter
	r    *http.Request
	cf   TransCipher
	pd   PDecoder
}

func (c *webconn) Read(p []byte) (n int, err error) {
	return c.data.Read(p)
}

func (c *webconn) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c *webconn) Close() error {
	return nil
}

func (c *webconn) TransCipher() TransCipher {
	return c.cf
}

func (c *webconn) Decoder() PDecoder {
	return c.pd
}

func (c *webconn) PeerAddr() string {
	return c.r.RemoteAddr
}

func (c *webconn) Header() http.Header {
	return c.w.Header()
}

func (c *webconn) WriteHeader(status int) {
	c.w.WriteHeader(status)
}
