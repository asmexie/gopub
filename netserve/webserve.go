package netserve

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/muroachanf/gopub/common"
	"bitbucket.org/muroachanf/gopub/netutils"
	"github.com/muroachanf/go-logger/logger"
	"github.com/muroachanf/mux"
)

var epoch = time.Unix(0, 0).Format(time.RFC1123)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

// WebServe ...
type WebServe struct {
	*mux.Router
	config *WebServeConfig
}

// GetQueryStrVar ...
func GetQueryStrVar(r *http.Request, key string) string {
	if r.Method == "GET" {
		return r.URL.Query().Get(key)
	}
	return r.FormValue(key)
}

// GetQueryFloatVar ...
func GetQueryFloatVar(r *http.Request, key string) float64 {
	s := GetQueryStrVar(r, key)
	if s == "" {
		panic(fmt.Errorf("can not find param %v in http request", key))
	}
	v, err := strconv.ParseFloat(s, 64)
	common.CheckError(err)
	return v
}

// GetQueryIntVar ...
func GetQueryIntVar(r *http.Request, key string) int {
	s := GetQueryStrVar(r, key)
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	common.CheckError(err)
	return v
}

// OnlyLocal ...
func OnlyLocal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		ips := strings.Split(r.Host, ":")
		if nil == net.ParseIP(ips[0]) {
			http.Error(w, "internal error", 500)
			return
		}
		if strings.HasPrefix(r.Host, "172.16.") ||
			strings.HasPrefix(r.Host, "192.168.") {
			h.ServeHTTP(w, r)
			return
		}
		http.Error(w, "internal error", 500)
	})
}

// NoCache ...
func NoCache(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}

		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// ServeHTTP ...
func (s *WebServe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if x := recover(); x != nil {
			logger.Error(r.URL.String(), x)
			if r.Host[:9] == "localhost" {
				http.Error(w, x.(error).Error(), http.StatusInternalServerError)
			} else {
				http.Error(w, "internal error", 500)
			}
		}
	}()
	s.Router.ServeHTTP(w, r)
}

// OnlyLocal ...
func (s *WebServe) OnlyLocal(f http.Handler) http.Handler {
	return OnlyLocal(f)
}

// NoCache ...
func (s *WebServe) NoCache(f http.Handler) http.Handler {
	return NoCache(f)
}

// NoCacheFunc ...
func (s *WebServe) NoCacheFunc(f http.HandlerFunc) http.Handler {
	return NoCache(f)
}

// RemoteIP ...
func RemoteIP(req *http.Request) string {
	return netutils.RemoteIP(req)
}

// WriteJSON ...
func (s *WebServe) WriteJSON(w http.ResponseWriter, data interface{}) error {
	return netutils.WriteJSON(w, data)
}

// GetQueryIntVar ...
func (s *WebServe) GetQueryIntVar(r *http.Request, key string) int {
	return GetQueryIntVar(r, key)
}

// GetQueryStrVar ...
func (s *WebServe) GetQueryStrVar(r *http.Request, key string) string {
	return GetQueryStrVar(r, key)
}

// Run ...
func (s *WebServe) Run(handler http.Handler) {
	logger.Info("new webserver on " + s.config.Port)
	server := &http.Server{
		Addr:           ":" + s.config.Port,
		Handler:        handler,
		ReadTimeout:    time.Duration(s.config.ReadTimeOut) * time.Second,
		WriteTimeout:   time.Duration(s.config.WriteTimeOut) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if s.config.CertPath == "" {
		logger.Fatal(server.ListenAndServe())
	} else {
		logger.Fatal(server.ListenAndServeTLS(s.config.CertPath, s.config.KeyPath))
	}
}

// NewWebServe ...
func NewWebServe(config *WebServeConfig) WebServe {
	s := WebServe{
		Router: mux.NewRouter(),
	}
	s.config = config
	return s
}
