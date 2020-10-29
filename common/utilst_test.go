package common

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/muroachanf/go-logger/logger"
)

func TestGetExecPath(t *testing.T) {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	logger.Debug(filepath.Join(filepath.Dir(path), "config.json"))
}

func TestReadIntf(t *testing.T) {
	testcfg := `
	{
		"base_server":{
			"port":22,
			"user":"root",
			"key_path":"vps.pri.pem"
		},
		"base_pserver": {
			"__base_obj_name__":"base_server",
			"use_proxy":true, 
			"proxy_addr":{"addr":"172.16.6.33","port":8061}
		},
		"base_db":{
			"port":3307,
			"user":"root"
		},
		"sv5_heart2_up_vultr":{
			"__base_obj_name__":"base_pserver",
			"host":"80.240.19.164",
			"db":{
				"__base_obj_name__":"base_db",
				"pwd":"N1IsjEpXgwll6DyMOQet",
				"dbname":"adcenter_yy",
				"appuser":"adhtusr",
				"apppwd":"PBhRvrawqSv24WhKldlr"
			}
		},
		"sv6_heart_but_germany_vultr":{
			"__base_obj_name__":"base_pserver",
			"host":"217.163.30.213",
			"db": {
				"__base_obj_name__":"base_db",
				"pwd":"JUnxbBQLDvo6OpiOqh0f",
				"dbname":"adyy_center",
				"appuser":"tinyhtusr",
				"apppwd":"SpI22aA74i0OPwKWj24X"
			}
		},
		"sv_heart_list":[
			"sv5_heart2_up_vultr","sv6_heart_but_germany_vultr"
		]
	}
	`

	cfg := ReadMap(strings.NewReader(testcfg), true)
	logger.Debugf("%v\n", cfg)
}

func TestEnvSubst(t *testing.T) {
	s := "${Test1} ${Test2:?test}"
	v, err := EnvSubstMap(s, (Map{"Test1": 3}))
	CheckError(err)
	logger.Debug(v)

	v, err = EnvSubst(s, (Map{"Test1": 3}).GetEnvvar)
	CheckError(err)
	logger.Debug(v)
}
