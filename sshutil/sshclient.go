package sshutil

import (
	"context"
	"net"
	"time"

	"github.com/muroachanf/go-logger/logger"
	"golang.org/x/crypto/ssh"
)

// SSHClient ...
type SSHClient struct {
	*ssh.Client
	ctx context.Context
}

// NewPortForward ...
func (sc *SSHClient) NewPortForward(localAddr, remoteAddr string) *PortForward {
	return NewPortForward(sc.ctx, sc.Dial, localAddr, remoteAddr)
}

// NewSocks5Proxy ...
func (sc *SSHClient) NewSocks5Proxy(localPort int, allowRemote bool) *Socks5Proxy {
	return NewSock5Proxy(sc.ctx, localPort, allowRemote, sc.Dial)
}

// Dial ...
func (sc *SSHClient) Dial(network, address string) (conn net.Conn, err error) {
	retryCnt := 1
	for {
		conn, err = sc.Client.Dial(network, address)
		if err == nil {
			return
		}

		logger.Debugf("Dial failed %v:%v %v", network, address, err)
		time.Sleep(time.Second * time.Duration(retryCnt))
		retryCnt++
		if retryCnt > 5 {
			break
		}

	}

	return
}

// NewSSHClient ...
func NewSSHClient(ctx context.Context, config SSHConfig, keypassFunc GetKeyPassFunc) (client *SSHClient, err error) {
	dialer, err := newSSHDialer(ctx, keypassFunc, config)
	if err != nil {
		return
	}
	client, err = dialer.newClient()

	return
}
