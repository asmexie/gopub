package sshutil

import (
	"context"
	"encoding/json"
	"testing"

	"bitbucket.org/muroachanf/gopub/common"
)

const testServer = `
{
	"port":22,
	"user":"root",
	"key_path":"D:\\project\\adcenter\\script\\ssh\\vps.pri.pem",
	"host":"68.183.214.189",
	"host_key":"AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBJ6eCbVbcXanQC0SoM10ex1PR6xWu3/8mspLK0Nriu1YWeTFgZ9kh9jwUyAEBzgqISzWXTb5mx+obHoyHdaXAyI=",
	"proxylist":"sv10_proxy_list"
}
`

func TestSSHClient(t *testing.T) {
	sshcfg := SSHConfig{}
	common.CheckError(json.Unmarshal([]byte(testServer), &sshcfg))

	client, err := NewSSHClient(context.Background(), sshcfg, func(keyPath string) string {
		return ""
	})
	common.CheckError(err)
	client.Close()
}
