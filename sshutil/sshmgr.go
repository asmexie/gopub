package sshutil

import (
	"context"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/go-errors/errors"
	"github.com/muroachanf/go-logger/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

// ProxyConfig ...
type ProxyConfig struct {
	SSHList     []SSHConfig
	AllowRemote bool
	ProxyPort   int
}

// SSHDialer ...
type SSHDialer struct {
	sshClientDialFunc  func() (*ssh.Client, error)
	waitSeconds        int64
	lastestConnectTime int64
	ctx                context.Context
	locker             sync.Mutex
}

func (sd *SSHDialer) newClient() (*SSHClient, error) {
	sd.locker.Lock()
	defer sd.locker.Unlock()

	now := time.Now().Unix()
	if sd.lastestConnectTime != 0 && now-sd.lastestConnectTime < sd.waitSeconds {
		return nil, nil
	}
	sd.lastestConnectTime = now
	client, err := sd.sshClientDialFunc()
	if err != nil {
		if sd.waitSeconds <= 0 {
			sd.waitSeconds = 5
		} else {
			sd.waitSeconds *= 2
		}
		common.LogError(err)
		return nil, err
	}

	sd.waitSeconds = 0
	return &SSHClient{
		Client: client,
		ctx:    sd.ctx,
	}, nil
}

func newSSHDialer(ctx context.Context, keypassFunc GetKeyPassFunc, config SSHConfig) (sd *SSHDialer, err error) {
	sd = &SSHDialer{ctx: ctx}
	sd.sshClientDialFunc, err = config.newSSHDialFunc(keypassFunc)
	if err != nil {
		sd = nil
		common.LogError(err)
		return
	}

	return
}

// SSHDialersMgr ...
type SSHDialersMgr struct {
	sshDialers       chan *SSHDialer
	ctx              context.Context
	config           ProxyConfig
	sshClient        *SSHClient
	proxyAddr        string
	testDialer       DialFunc
	connIsValid      bool
	lastCheckUrlTime int64
}

// NewSSHDialersMgr ...
func NewSSHDialersMgr(ctx context.Context, keypassFunc GetKeyPassFunc, config ProxyConfig) *SSHDialersMgr {
	logger.Debugf("got config:%v", config)
	sm := &SSHDialersMgr{
		ctx:    ctx,
		config: config,
	}
	sm.Init(keypassFunc)
	return sm
}

// Init ...
func (sds *SSHDialersMgr) Init(keypassFunc GetKeyPassFunc) {
	if len(sds.config.SSHList) == 0 {
		return
	}
	sds.sshDialers = make(chan *SSHDialer, len(sds.config.SSHList))
	for _, proxyConfig := range sds.config.SSHList {
		d, err := newSSHDialer(sds.ctx, keypassFunc, proxyConfig)
		if err != nil {
			common.LogError(err)
			continue
		}
		sds.sshDialers <- d
	}
	if len(sds.sshDialers) == 0 {
		return
	}
	go sds.StartServe()
}

func (sds *SSHDialersMgr) popSSHDialer() (dialer *SSHDialer) {
	select {
	case dialer = <-sds.sshDialers:
		return
	default:
		return
	}
}

func (sds *SSHDialersMgr) connetToSSH() (*SSHClient, bool) {
	sshClients := make(chan *SSHClient, cap(sds.sshDialers))
	sshClient := make(chan *SSHClient, 1)
	dialers := []*SSHDialer{}
	for {
		d := sds.popSSHDialer()
		if d == nil {
			break
		}

		dialers = append(dialers, d)

		go func(d *SSHDialer) {
			if client, err := d.newClient(); client != nil && err == nil {
				sshClients <- client
			} else {
				sshClients <- nil
			}
			sds.sshDialers <- d
		}(d)
	}
	go func() {
		haveGet := false
		for i := 0; i < len(dialers); i++ {
			client := <-sshClients
			if client != nil {
				if !haveGet {
					sshClient <- client
					haveGet = true
				} else {
					client.Close()
				}
			}
		}

		if !haveGet {
			logger.Debug("not connect to any valid sshclient")
			sshClient <- nil
		} else {
			logger.Debug("connect to sshclient success")
		}
	}()
	select {
	case client := <-sshClient:
		return client, false
	case <-time.After(time.Second * 20):
		return nil, false
	case <-sds.ctx.Done():
		return nil, true
	}
}

func (sds *SSHDialersMgr) serverIsOK() bool {
	if sds.connIsValid {
		return true
	}

	if !sds.checkSSHState() {
		return false
	}

	now := time.Now().Unix()
	if now-sds.lastCheckUrlTime < 10 {
		return true
	}
	sds.lastCheckUrlTime = now
	if !sds.testGetWebs() {
		logger.Debugf("check ssh %v https state failed", sds.sshClient.RemoteAddr().String())
		return false
	}

	return true
}

// StartServe ...
func (sds *SSHDialersMgr) StartServe() {
	defer func() {
		logger.Debug("Serve over")
		if x := recover(); x != nil {
			log.Println(errors.Wrap(x, 0).ErrorStack())
		}
	}()
	sds.lastCheckUrlTime = time.Now().Unix()
	for {
		sds.connIsValid = sds.serverIsOK()
		if !sds.connIsValid {
			if sds.sshClient != nil {
				sds.sshClient.Close()
			}
			sshClient, terminate := sds.connetToSSH()
			if terminate {
				return
			}
			sds.sshClient = sshClient
		}
		if !common.Sleep(sds.ctx, time.Second*2) {
			return
		}
	}

}

func (sds *SSHDialersMgr) testDial(url string) (isok bool) {
	defer func() {
		logger.Debugf("test dial @%v url %v result %v", sds.proxyAddr, url, isok)
	}()
	if sds.testDialer == nil {
		return true
	}
	res := make(chan bool)
	go func() {
		tr := &http.Transport{
			Dial:                  sds.testDialer,
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		client := &http.Client{Transport: tr}
		resp, err := client.Get("https://" + url)
		if err != nil {
			common.LogError(err)
			res <- false
			return
		}
		if resp.Body != nil {
			defer resp.Body.Close()
		}

		res <- true
	}()
	select {
	case isok = <-res:
		return
	case <-time.After(time.Second * 30):
		return
	}
}

func (sds *SSHDialersMgr) checkSSHState() bool {
	sshClient := sds.sshClient
	if sshClient != nil {
		_, _, err := sshClient.SendRequest("keepalive@golang.org", true, nil)
		if err != nil {
			logger.Debugf("check socks %v ssh %v ping state failed", sds.proxyAddr, sshClient.RemoteAddr().String())
			common.LogError(err)
			return false
		}
		return true
	}
	return false
}

func (sds *SSHDialersMgr) testGetWebs() bool {
	testDomains := []string{"www.microsoft.com", "twitter.com", "www.google.com", "www.facebook.com", "www.apple.com"}

	randIdx := int(rand.Int31()) % len(testDomains)
	if sds.testDial(testDomains[randIdx]) {
		return true
	}
	for i := 0; i < len(testDomains); i++ {
		if i != randIdx {
			if sds.testDial(testDomains[i]) {
				return true
			}
		}
	}
	return false
}

// StartProxy ...
func (sds *SSHDialersMgr) StartProxy() {
	logger.Debugf("start proxy to proxy at port %v", sds.config.ProxyPort)
	s := NewSock5Proxy(sds.ctx, sds.config.ProxyPort, sds.config.AllowRemote,
		sds.GetDialFunc())
	s.Start()

	addr, err := s.Addr()
	common.CheckError(err)

	sds.proxyAddr = addr
	dialer, err := proxy.SOCKS5("tcp", addr,
		nil,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)

	common.CheckError(err)
	sds.testDialer = dialer.Dial

	logger.Debugf("start proxy at %v", addr)
}

// GetDialFunc ...
func (sds *SSHDialersMgr) GetDialFunc() DialFunc {
	return func(network, address string) (net.Conn, error) {
		if sds.sshClient != nil {
			conn, err := sds.sshClient.Dial(network, address)
			if err != nil {
				sds.connIsValid = false
			}
			return conn, err
		}
		return nil, errors.New("not found valid ssh client")
	}
}
