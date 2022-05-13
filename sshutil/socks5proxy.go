package sshutil

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/muroachanf/go-logger/logger"
	socks5 "github.com/muroachanf/go-socks5"
)

var netListen = net.Listen

// DialFunc ...
type DialFunc func(network, address string) (net.Conn, error)

// Socks5Proxy ...
type Socks5Proxy struct {
	port    int
	remote  bool
	started bool
	logger  *log.Logger
	mtx     sync.Mutex
	ctx     context.Context
	dialer  DialFunc

	netListener net.Listener
	terminated  chan bool
}

// NewSocks5Server ...
func (s *Socks5Proxy) NewSocks5Server() (*socks5.Server, error) {
	conf := &socks5.Config{
		Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.dialer(network, addr)
		},
		Logger: s.logger,
	}

	return socks5.New(conf)
}

// ListenAndServe ...
func (s *Socks5Proxy) ListenAndServe(network, addr string) error {
	server, err := s.NewSocks5Server()
	if err != nil {
		return err
	}
	l, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	defer l.Close()
	s.netListener = l
	waitSeconds := time.Second
	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-s.terminated:
				return nil
			default:
			}
			waitSeconds *= 2
			logger.Debugf("accept got error %v", err.Error())
			if !common.Sleep(s.ctx, waitSeconds) {
				return nil
			}
			continue
		}
		waitSeconds = time.Second
		go server.ServeConn(conn)
	}
}

func openPort() (int, error) {
	l, err := netListen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(port)
}

// Stop ...
func (s *Socks5Proxy) Stop() {
	s.terminated <- true
	if s.netListener != nil {
		s.netListener.Close()
	}
}

// Start ...
func (s *Socks5Proxy) Start() (err error) {

	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.port == 0 {
		s.port, err = openPort()
		if err != nil {
			return err
		}
	}

	go func() {
		addr := fmt.Sprintf(":%d", s.port)
		if !s.remote {
			addr = "127.0.0.1" + addr
		}

		if err := s.ListenAndServe("tcp", addr); err != nil {
			logger.Debugf("list server failed with error %v", err.Error())
		}

	}()

	s.started = true
	return nil
}

// Addr ...
func (s *Socks5Proxy) Addr() (string, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.port == 0 {
		return "", errors.New("socks5 proxy is not running")
	}
	addr := fmt.Sprintf(":%d", s.port)
	if !s.remote {
		addr = "127.0.0.1" + addr
	}
	return addr, nil
}

// NewSock5Proxy ...
func NewSock5Proxy(ctx context.Context, port int, allowRemoteCon bool, dialer DialFunc) *Socks5Proxy {
	return &Socks5Proxy{
		started:    false,
		logger:     nil,
		port:       port,
		terminated: make(chan bool, 1),
		remote:     allowRemoteCon,
		ctx:        ctx,
		dialer:     dialer,
	}
}
