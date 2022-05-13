package netutils

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/asmexie/gopub/common"
)

// ReadRespToMap ...
func ReadRespToMap(resp *http.Response) common.Map {
	var data interface{}
	err := json.NewDecoder(resp.Body).Decode(&data)
	common.CheckError(err)
	return data.(map[string]interface{})
}

// GenGetURL ...
func GenGetURL(reqPath string, params URLValues) string {
	return reqPath + "?" + params.Encode()
}

// MapToURLValues ...
func MapToURLValues(reqParams map[string]interface{}) URLValues {
	values := MakeURLValues(nil)
	for k, v := range reqParams {
		values.Set(k, fmt.Sprintf("%v", v))
	}
	return values
}

// EncodeURL ...
func EncodeURL(reqURL string, reqParams map[string]interface{}) string {
	return reqURL + "?" + MapToURLValues(reqParams).Encode()
}
