package sshutil

import (
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/go-logger/logger"
)

// PortForward ...
type PortForward struct {
	ctx        context.Context
	dialFunc   DialFunc
	localAddr  string
	remoteAddr string
	terminated chan bool
	ls         net.Listener
}

// NewPortForward ...
func NewPortForward(ctx context.Context, dialFunc DialFunc, localAddr, remoteAdrr string) *PortForward {

	if localAddr == "" {
		localPort, err := openPort()
		common.CheckError(err)
		localAddr = "127.0.0.1:" + strconv.Itoa(localPort)
	}
	return &PortForward{
		ctx:        ctx,
		dialFunc:   dialFunc,
		localAddr:  localAddr,
		remoteAddr: remoteAdrr,
		terminated: make(chan bool),
	}
}

// LocalPort ...
func (f *PortForward) LocalPort() string {
	localAddr := strings.Split(f.LocalAddr(), ":")
	if len(localAddr) <= 1 {
		return ""
	}
	return localAddr[1]
}

// LocalAddr ...
func (f *PortForward) LocalAddr() string {
	return f.localAddr
}

// Stop ...
func (f *PortForward) Stop() {
	close(f.terminated)
	f.ls.Close()

}

// Start ...
func (f *PortForward) Start() {
	f.ListenAndServe()
}

func copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}

func (f *PortForward) serveConn(conn net.Conn) {
	logger.Debugf("start serveConn")
	rconn, err := f.dialFunc("tcp", f.remoteAddr)
	common.CheckError(err)

	go copyIO(conn, rconn)
	go copyIO(rconn, conn)

}

// ListenAndServe ...
func (f *PortForward) ListenAndServe() error {
	logger.Debugf("start listen portforward at %v", f.localAddr)
	l, err := net.Listen("tcp", f.localAddr)
	if err != nil {
		return err
	}

	f.ls = l
	defer l.Close()
	waitSeconds := time.Second
	for {
		conn, err := l.Accept()
		select {
		case <-f.terminated:
			return err
		default:
		}
		if err != nil {
			waitSeconds *= 2
			common.LogError(err)
			if !common.Sleep(f.ctx, waitSeconds) {
				return nil
			}
			continue
		}
		waitSeconds = time.Second
		go f.serveConn(conn)
	}
}
