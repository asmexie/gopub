package netserve

import (
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/go-logger/logger"
)

func TestRSAKey(t *testing.T) {
	prikey := "MIICWwIBAAKBgQCdbPJ8Banzv43RH59Konx9llqsy6PgI+/DkJuJki7VglV4BeDQNnuuUD4eMse5hNm7TL05H5UprJJSm4lCdSUcPdKTCCrstlCrM8qw+tNiBNMBPGh+9KZf1Tl9tqcHa7xM267w6mHlO7VV3A5cchAZDILHD/2cq/qd8TxZG9vpJwIDAQABAoGAM5IGGYTNePEOZyxhxVRXTdjcWXDYfUuodrs/iKCfwQfSMeBTFkJS3/afcssVzHttzELGVhk3hxBmWrNjEqdHgWZKD3wTPLrY2Kpd8+1V+ioYJBRlS4iD6DIp5KzMuXkic43lNdRd6OpQgJLxDPF9FkcWUPIe8XZhvONuPphVR5ECQQDQyHiIDthc58ljN54fnOzhgY7pye6/1lRgrcqyhc/VtKiqCk4MbeJboUncFQR9e1JZ3vdOzJW/fk1IU9YVywm/AkEAwQcgCyYZHi/5TYVAAnrrkWYAc9LgH9UzDfkR1z4O9kto/4ph6L9l/42aarlApi3ryrUsOKxKytI/1TFqdwZqmQJAarwR4ny0X8qfSfnE/KRc9Wwmg56YT7piqIowddOyzK3vC/74p6IFdpKeD8Uu5neFQiyagc5VP/Bx0egKKloCQQJATZj5rsGwE0yh4iIRK24SyS7CO82oP+PLVHCuVWMjTKvgF+qflZtr+6IHU6QJc0S+p4zRrC7HGmYPNztYW2T+8QJAf9UN8Inwy71AUFHE1cBgcEMRCLV5LG/jnsrklWSx/5PdLPsDVm9OpccVthN4O/a8FrOv4nqYIBsWMdSfKjjADQ=="
	key, err := base64.StdEncoding.DecodeString(prikey)
	common.CheckError(err)
	rsaKey, err := x509.ParsePKCS1PrivateKey(key)
	common.CheckError(err)
	publicKey := rsaKey.PublicKey
	data, err := x509.MarshalPKIXPublicKey(&publicKey)
	common.CheckError(err)
	logger.Debugf("got public key:% x", data)
}
