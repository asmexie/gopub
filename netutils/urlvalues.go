package netutils

import (
	"fmt"
	"net/url"
	"strconv"
)

// URLValues ...
type URLValues url.Values

// MakeURLValues ...
func MakeURLValues(values url.Values) URLValues {
	if values == nil {
		values = make(url.Values)
	}
	return URLValues(values)
}

// Set ...
func (uv URLValues) Set(key, value string) {
	url.Values(uv).Set(key, value)
}

// Get ...
func (uv URLValues) Get(key string) string {
	return url.Values(uv).Get(key)
}

// StrValue ...
func (uv URLValues) StrValue(key string) string {
	return url.Values(uv).Get(key)
}

// IntValue ...
func (uv URLValues) IntValue(key string, defValue int) int {
	s := uv.StrValue(key)
	if s == "" {
		return defValue
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defValue
	}
	return v
}

// FloatValue ...
func (uv URLValues) FloatValue(key string, defValue float64) float64 {
	s := uv.StrValue(key)
	if s == "" {
		panic(fmt.Errorf("can not find param %v in http request", key))
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return defValue
	}
	return v
}

// Add ...
func (uv URLValues) Add(key, value string) {
	url.Values(uv).Add(key, value)
}

// Encode ...
func (uv URLValues) Encode() string {
	return url.Values(uv).Encode()
}
