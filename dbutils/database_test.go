package dbutils

import (
	"encoding/json"
	"fmt"
	"testing"

	"bitbucket.org/muroachanf/gopub/common"
)

func TestGenCUID(t *testing.T) {
	InitCuidDev(0x01)
	for i := 0; i < 10; i++ {
		cuid := NewCUID()
		fmt.Printf("got new cuid:%d,%x\n", cuid, cuid)
		data, err := json.Marshal(cuid)
		common.CheckError(err)
		fmt.Println(string(data))
		common.CheckError(json.Unmarshal(data, &cuid))
		fmt.Println("got parsed cuid", cuid)

		mp := map[string]CUID{}
		mp["cuid"] = cuid
		data, err = json.Marshal(mp)
		common.CheckError(err)
		fmt.Println(string(data))
		mp = nil
		common.CheckError(json.Unmarshal(data, &mp))
		fmt.Println("got parsed cuid", mp["cuid"])
	}
}
