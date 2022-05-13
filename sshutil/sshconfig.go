package sshutil

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/muroachanf/go-logger/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

var priKeyPassMap = make(map[string]string)

// GetKeyPassFunc ...
type GetKeyPassFunc func(keyPath string) string

// SSHConfig ...
type SSHConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	UserName      string `json:"user"`
	KeyPath       string `json:"key_path"`
	ProxyAddrs    string `json:"proxy_addr"`
	HostKey       string `json:"host_key"`
	Timeout       int    `json:"time_out"`
	VerifyHostKey bool   `json:"verify_host_key"`
}

func testEq(a, b []byte) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func (p SSHConfig) readKeyData(keypassFunc GetKeyPassFunc) ([]byte, error) {
	key, err := ioutil.ReadFile(p.KeyPath)
	if err != nil {
		return nil, err
	}

	md5Hash := common.CalcDataMd5(key)

	block, rest := pem.Decode(key)
	if len(rest) > 0 {
		return nil, common.ERR(errors.New("decode rsa private key failed"))
	}
	keypass, ok := priKeyPassMap[md5Hash]
	if !ok {
		keypass = keypassFunc(p.KeyPath)
	}

	der, err := x509.DecryptPEMBlock(block, []byte(keypass))
	if err != nil {
		return nil, common.ERR(err)
	}
	priKeyPassMap[md5Hash] = keypass
	return der, err
}

// SSHClientConfig ...
func (p SSHConfig) SSHClientConfig(keyPassFunc GetKeyPassFunc) (*ssh.ClientConfig, error) {
	username := p.UserName
	if username == "" {
		username = "jumpbox"
	}
	logger.Debug("using name:", username)

	keyData, err := p.readKeyData(keyPassFunc)
	if err != nil {
		logger.Debug("read key failed")
		return nil, err
	}

	priKey, err := x509.ParsePKCS1PrivateKey([]byte(keyData))
	if err != nil {
		logger.Debug("parse prikey failed")
		return nil, err
	}

	signer, err := ssh.NewSignerFromKey(priKey)
	if err != nil {
		logger.Debug("NewSignerFromKey failed")
		return nil, err
	}
	hostKey, err := base64.StdEncoding.DecodeString(p.HostKey)
	if err != nil {
		common.LogError(err)
		return nil, err
	}

	return &ssh.ClientConfig{
		Timeout: 30 * time.Second,
		User:    username,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			//logger.Debugf("host key callback at host %v", hostname)
			if (hostKey != nil && len(hostKey) > 0) && testEq(key.Marshal(), hostKey) {
				return nil
			}
			if hostKey == nil || len(hostKey) == 0 {
				if !p.VerifyHostKey {
					return nil
				}

			}
			fmt.Printf("invalid host %v key:%v\n", hostname, base64.StdEncoding.EncodeToString(key.Marshal()))
			return fmt.Errorf("host %v key is invalid", hostname)
		},
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
	}, nil
}

func (p SSHConfig) newSSHDialFunc(keypassFunc GetKeyPassFunc) (func() (*ssh.Client, error), error) {
	cfg, err := p.SSHClientConfig(keypassFunc)
	if err != nil {
		return nil, err
	}
	return func() (*ssh.Client, error) {
		url := fmt.Sprintf("%v:%v", p.Host, p.Port)
		return p.DialSSH("tcp", url, cfg)
	}, nil
}

// Get ...
func (p SSHConfig) Get(username, privateKey, serverURL string) (ssh.PublicKey, error) {
	logger.Debugf("get public key failed")
	return nil, nil
}

func (p SSHConfig) timeout() time.Duration {
	if p.Timeout == 0 {
		return time.Second * 30
	}
	return time.Second * time.Duration(p.Timeout)
}

// GetDialConn ...
func (p SSHConfig) GetDialConn(network, addr, proxyAddr string) (net.Conn, error) {
	timeout := p.timeout()
	if proxyAddr == "" {
		return net.DialTimeout(network, addr, timeout)
	}
	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyAddr, nil, &net.Dialer{
		Timeout:   timeout,
		KeepAlive: timeout,
	})
	if err != nil {
		return nil, err
	}
	return dialSocksProxy.Dial("tcp", addr)
}

// DialSSH ...
func (p SSHConfig) DialSSH(network, addr string, clientConfig *ssh.ClientConfig) (*ssh.Client, error) {
	dialSSH := func(proxyAddr string) (*ssh.Client, error) {
		logger.Debugf("start connect to %v by proxy %v", addr, proxyAddr)
		conn, err := p.GetDialConn(network, addr, proxyAddr)
		if err != nil {
			return nil, err
		}
		timeoutConn := &TimeoutConn{conn, p.timeout(), p.timeout()}
		c, chans, reqs, err := ssh.NewClientConn(timeoutConn, addr, clientConfig)
		if err != nil {
			return nil, err
		}

		return ssh.NewClient(c, chans, reqs), nil
	}

	proxyAddrs := strings.Split(p.ProxyAddrs, ";")
	if len(proxyAddrs) == 0 {
		return dialSSH("")
	}

	for i := 0; i < 3; i++ {
		for _, proxyAddr := range proxyAddrs {
			client, err := dialSSH(proxyAddr)
			if err == nil {
				return client, err
			}
		}
	}
	return nil, fmt.Errorf("connect to %v failed", addr)
}

// HostMapToSSHConfig ...
func HostMapToSSHConfig(host common.MapIntf) (config SSHConfig) {
	keyDir := host.Str("keyDir")
	keyPath := host.Str("keyPath")
	if keyPath == "" {
		keyPath = "id_rsa"
	}
	keyPath = filepath.Join(keyDir, keyPath)
	config.KeyPath = keyPath
	config.Host = host.Str("host")
	config.Port = host.Int("port")
	config.ProxyAddrs = host.Str("proxy")
	config.UserName = host.Str("user")
	config.Timeout = host.Int("timeout")
	config.HostKey = host.Str("hostkey")
	config.VerifyHostKey = config.HostKey != ""
	return
}
