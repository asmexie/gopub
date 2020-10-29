package common

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/muroachanf/envsubst"

	"github.com/termie/go-shutil"
	"gopkg.in/yaml.v2"

	"github.com/go-errors/errors"
	"github.com/muroachanf/go-logger/logger"
)

// CheckError panic when error happened
func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

// CheckPanicAndLogErr panic when error happened
func CheckPanicAndLogErr(e error) {
	if e != nil {
		LogError(e)
		panic(e)
	}
}

// ERR ...
func ERR(err error) error {
	return LogError(err)
}

// LogPanicErr ...
func LogPanicErr() {
	if e := recover(); e != nil {
		logger.Debugf("err:%v\nstacktrace:\n%v", e.(error), string(debug.Stack()))
	}
}

// CheckPanicErr ...
func CheckPanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// LogERR ...
func LogERR(err error) error {
	return LogError(err)
}

// ErrStack ...
func ErrStack(err error) string {
	return err.Error() + "\n" + string(debug.Stack())
}

// LogError write error info and stack trace into log file.
func LogError(e interface{}) error {
	if e == nil {
		return nil
	}

	logger.ErrorN(1, errors.Wrap(e, 1).ErrorStack())
	return e.(error)
}

// InitContext ...
func InitContext(logfile string) (context.Context, context.CancelFunc) {
	var f *os.File
	var err error
	if logfile != "" {
		f, err = os.OpenFile(logfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	}

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt, os.Kill)
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		<-stopChan // wait for SIGINT
		if logfile != "" {
			f.Close()
		}
		cancelFunc()
	}()
	return ctx, cancelFunc
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandStringRunes ...
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// RandBytes ...
func RandBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return b
}

// WriteJSON ...
func WriteJSON(w io.Writer, obj interface{}) {
	CheckError(json.NewEncoder(w).Encode(obj))
}

// ReadJSON ...
func ReadJSON(r io.Reader, obj interface{}) {
	CheckError(json.NewDecoder(r).Decode(obj))
}

// JsonToObj ...
func JsonToObj(data []byte, obj interface{}) {
	err := json.Unmarshal(data, obj)
	CheckError(err)
}

var _execDir = ""

// GetExecDir ...
func GetExecDir() string {
	if _execDir == "" {
		file, _ := exec.LookPath(os.Args[0])
		path, _ := filepath.Abs(file)
		_execDir = filepath.Dir(path)
	}
	return _execDir
}

// ToOSPath ...
func ToOSPath(path string) string {
	if filepath.Separator == '/' {
		return strings.Replace(path, "\\", "/", -1)
	}
	return strings.Replace(path, "/", "\\", -1)
}

// ReadConfigData ...
func ReadConfigData(configPath string) ([]byte, error) {
	if configPath == "" {
		configPath = filepath.Join(GetExecDir(), "config.json")
		logger.Debug(configPath)
	}

	return ioutil.ReadFile(configPath)
}

// DumpHttpRequest ...
func DumpHttpRequest(r *http.Request, w io.Writer, dumpBody bool) {
	v, err := httputil.DumpRequest(r, dumpBody)
	CheckError(err)
	fmt.Fprint(w, string(v))
}

// DumpHttpResponse ...
func DumpHttpResponse(resp *http.Response, w io.Writer, dumpBody bool) {
	v, err := httputil.DumpResponse(resp, dumpBody)
	CheckError(err)
	fmt.Fprint(w, string(v))
}

// CheckArgsSize ...
func CheckArgsSize(args []string, size int) {
	if len(args) != size {
		panic(fmt.Errorf("arg  <<%s>> size not equal %d", strings.Join(args, " "), size))
	}
}

// NowDateTimeStr return current date and time, format as string
func NowDateTimeStr() (string, string) {
	now := time.Now()
	return now.Format("2006-01-02"), now.Format("15:04:05")

}

// FileExists check the file if exists, if not exits, return false and fail error
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// InitLogger ...
func InitLogger(logDir, logFile string) {
	logger.SetLevel(logger.DEBUG)

	if logDir == "" {
		logDir = filepath.Join(GetExecDir(), "log")
	}
	if b, _ := FileExists(logDir); !b {
		os.Mkdir(logDir, os.ModeDir)
	}
	if b, _ := FileExists(logDir); b {
		logger.SetRollingDaily(logDir, logFile)
	}
}

// Wait ...
func Wait(c <-chan struct{}, d time.Duration) bool {
	select {
	case <-c:
		return false
	case <-time.After(d):
		return true
	}
}

// MapIntf ...
type MapIntf interface {
	Value(string) (v interface{}, ok bool)
}

// StringMap ...
type StringMap map[string]string

// Value ...
func (m StringMap) Value(key string) (v interface{}, ok bool) {
	v, ok = m[key]
	return
}

const cVarExprStr string = "\\{\\{\\.\\w+\\}\\}"

var varExprRegexp = regexp.MustCompile(cVarExprStr)

// ExpandVarExpr will expand the variables which the format like '{{.var}}' in a string with special context
func ExpandVarExpr(expr string, context MapIntf) string {
	return varExprRegexp.ReplaceAllStringFunc(expr, func(repl string) string {
		varName := repl[3 : len(repl)-2]
		if v, ok := context.Value(varName); ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	})
}

// IVarReExpr ...
type IVarReExpr interface {
	ParseVarName(varExpr string) string
	GetRegexpr() *regexp.Regexp
}

type varReExpr2 struct{}

func (r varReExpr2) ParseVarName(s string) string {
	return s[4 : len(s)-2]
}

const cVarExprStr2 string = "\\$\\{\\{\\.\\w+\\}\\}"

var varExprRegexp2 = regexp.MustCompile(cVarExprStr2)

func (r varReExpr2) GetRegexpr() *regexp.Regexp {
	return varExprRegexp2
}

var vre2 varReExpr2

// GetVarReExpr ...
func GetVarReExpr(vetype int) IVarReExpr {
	return vre2
}

// GetVarValueProc ...
type GetVarValueProc func(varName string) (interface{}, bool)

// ExpandVarExprEx will expand the variables which the format like '{{.var}}' in a string with special context
func ExpandVarExprEx(varExpr IVarReExpr, expr string, f GetVarValueProc) string {
	return varExpr.GetRegexpr().ReplaceAllStringFunc(expr, func(repl string) string {
		varName := varExpr.ParseVarName(repl)
		if v, ok := f(varName); ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	})
}

// StrToTime ...
func StrToTime(t string) time.Time {
	tm, err := time.ParseInLocation("2006-01-02 15:04:05", t, time.Local)
	if err == nil {
		return tm
	}
	tm, err = time.ParseInLocation("2006-01-02", t, time.Local)
	if err == nil {
		return tm
	}
	tm, err = time.ParseInLocation("15:04:05", t, time.Local)
	CheckError(err)

	return tm
}

// DateToStr ...
func DateToStr(t time.Time) string {
	return t.Format("2006-01-02")
}

// TimeToStr ...
func TimeToStr(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// CopyMap ...
func CopyMap(src, dest map[string]interface{}) {
	for rk, rv := range src {
		dest[rk] = rv
	}
}

func getIntefaceIps(i net.Interface) []net.IP {
	ips := []net.IP{}
	addrs, err := i.Addrs()
	if err != nil {
		logger.Error(err)
		return ips
	}
	// handle err
	for _, addr := range addrs {
		switch v := addr.(type) {
		case *net.IPNet:
			ips = append(ips, v.IP)
		case *net.IPAddr:
			ips = append(ips, v.IP)
		}
		// process IP address
	}
	return ips
}

// LocalIPs ...
func LocalIPs() []net.IP {
	ifaces, err := net.Interfaces()
	CheckError(err)
	// handle err
	ips := []net.IP{}
	for _, i := range ifaces {

		if err != nil {
			logger.Error(err)
			continue
		}
		ips = append(ips, getIntefaceIps(i)...)

	}
	return ips
}

// TimerAction ...
func TimerAction(ctx context.Context, d time.Duration, actionName string, action func()) {
	defer logger.Debug("TimerAction " + actionName + "over")
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		func() {
			defer LogPanicErr()
			action()
		}()

		select {
		case <-ctx.Done():
			return
		case <-time.After(d):
		}
	}
}

// StrToInt64 ...
func StrToInt64(s string, defV int64) int64 {
	if s == "" {
		return defV
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		LogError(err)
		return defV
	}
	return v
}

// CalcDataMd5 ...
func CalcDataMd5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// CalcFileMd5 ...
func CalcFileMd5(fullFileName string) (string, error) {
	data, err := ioutil.ReadFile(fullFileName)
	if err != nil {
		return "", err
	}
	return CalcDataMd5(data), nil
}

// CopyFiles ...
func CopyFiles(srcdir, dstdir string, files []string) error {
	for _, fname := range files {
		err := shutil.CopyFile(filepath.Join(srcdir, fname), filepath.Join(dstdir, fname), false)
		if err != nil {
			if _, ok := err.(*shutil.SameFileError); !ok {
				return err
			}
		}
	}
	return nil
}

// RemoveDirContents ...
func RemoveDirContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// ExecCmd ...
func ExecCmd(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

// LoadYMLData ...
func LoadYMLData(data []byte, config interface{}) error {
	return yaml.Unmarshal(data, config)
}

// LoadYMLFile ...
func LoadYMLFile(configFile string, config interface{}) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	return LoadYMLData(data, config)
}

// LimitStr ...
func LimitStr(s string, size int) string {
	sn := []rune(s)
	if len(sn) > size {
		return string(sn[:size])
	}
	return s
}

// EnvSubstMap ...
func EnvSubstMap(s string, m map[string]interface{}) (string, error) {
	varMap := Map(m)
	return EnvSubst(s, func(varname string) (string, bool) {
		return varMap.Str(varname), true
	})
}

// EnvSubstOS ...
func EnvSubstOS(s string) string {
	v, err := EnvSubst(s, func(varName string) (string, bool) {
		return os.Getenv(varName), true
	})
	CheckError(err)
	return v
}

// EnvSubst ...
func EnvSubst(s string, mapping func(varName string) (string, bool)) (string, error) {

	s, err := envsubst.Eval(s, mapping)
	if err != nil {
		return "", err
	}
	return s, nil
}

// SSContains ...
func SSContains(a []string, x string) bool {
	x = strings.ToLower(x)
	for _, n := range a {
		if strings.ToLower(n) == x {
			return true
		}
	}
	return false
}

// DateIsEqual ...
func DateIsEqual(d1, d2 time.Time) bool {
	y1, m1, dd1 := d1.Date()
	y2, m2, dd2 := d2.Date()
	return y1 == y2 && m1 == m2 && dd1 == dd2
}

// InterfaceSlice ...
func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

// ShuffleInterfaceSlice ...
func ShuffleInterfaceSlice(slice []interface{}) []interface{} {

	if len(slice) <= 1 {
		return slice
	}

	shuffleSlice := make([]interface{}, len(slice))
	for n := 0; n < len(slice); n++ {
		shuffleSlice[n] = slice[n]
		if n == 0 {
			continue
		}
		swapSliceItem := func(sl []interface{}, a, b int) {
			v := sl[a]
			sl[a] = sl[b]
			sl[b] = v
		}
		if n == 1 {
			if rand.Int()%2 == 0 {
				swapSliceItem(shuffleSlice, 0, 1)
			}
		}
		k := rand.Int() % n
		swapSliceItem(shuffleSlice, k, n)
	}
	return shuffleSlice
}
