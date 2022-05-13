package netutils

import (
	"fmt"
	"net/url"

	"github.com/asmexie/gopub/common"
)

// MapURL ...
type MapURL struct {
	rawurl *url.URL
	host   string
	params url.Values
}

// ParseURL ...
func (m *MapURL) ParseURL(reqURL string) error {
	var err error
	m.rawurl, err = url.Parse(reqURL)
	m.params = m.rawurl.Query()
	return err
}

// AddMapParams ...
func (m *MapURL) AddMapParams(p common.Map) *MapURL {
	for k, v := range p {
		m.params.Set(k, fmt.Sprintf("%v", v))
	}
	return m
}

// Encode ...
func (m *MapURL) Encode() string {
	m.rawurl.RawQuery = m.params.Encode()
	return m.rawurl.String()
}

// AddParam ...
func (m *MapURL) AddParam(k string, v interface{}) *MapURL {
	m.params.Set(k, fmt.Sprintf("%v", v))
	return m
}

// ParseURL ...
func ParseURL(reqURL string) (*MapURL, error) {
	m := &MapURL{}
	return m, m.ParseURL(reqURL)
}
