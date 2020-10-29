package netserve

type NetServeConfig struct {
	Port         []int
	NetType      []string
	ListenIP     []string
	Cipher       []string
	CodeType     string
	CipherID     int
	LogVerbose   bool
	ReadTimeOut  int
	WriteTimeOut int
	Debug        int
	HandlerName  string
}

// WebServeConfig ...
type WebServeConfig struct {
	Port         string
	Host         []string
	PprofIps     []string
	ReadTimeOut  int
	WriteTimeOut int
	WebPath      string
	CertPath     string
	KeyPath      string
}

// APIHandler ...
type APIHandler interface {
	HandleAPI(conn SimpleNetConn, api int, data []byte)
	ConvertSApiToCode(apis string) int
	ConvertAPIToCode(api int) int
	QueryAppSecretKey(app string) string
}

// NameToAPIHandler ...
type NameToAPIHandler func(name string) APIHandler
