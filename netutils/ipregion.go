package netutils

import (
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/go-logger/logger"
)

// IncRegInfo ...
func IncRegInfo(region string, regionInfo map[string]int, count int) {
	// logger.Debugf("inc:%v,%v", region, count)
	if v, ok := regionInfo[region]; ok {
		regionInfo[region] = v + count
	} else {
		regionInfo[region] = count
	}
}

func ip2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

// CountryIPInfo ...
type CountryIPInfo struct {
	IPFrom  uint32 `db:"ip_from"`
	IPTo    uint32 `db:"ip_to"`
	Country string `db:"country"`
	A2      string `db:"a2"`
	A3      string `db:"a3"`
}

// CountryInfo ...
type CountryInfo struct {
	Country     string `db:"country"`
	A2          string `db:"a2"`
	A3          string `db:"a3"`
	Num         int    `db:"country_num"`
	DialingCode string `db:"cdcode"`
}

// IPRegions ...
type IPRegions struct {
	countryIPInfos  []CountryIPInfo
	countryInfosAA  map[string]*CountryInfo
	countryInfosNum map[int]*CountryInfo
}

var __ipr IPRegions

// IPR ...
func IPR() *IPRegions {
	return &__ipr
}

// CheckLoadResult ...
func CheckLoadResult(cnt int, err error) {
	common.CheckError(err)
}

func loadCSVFile(fileName string) [][]string {
	fi, err := os.Open(fileName)
	common.CheckError(err)
	defer fi.Close()
	res, err := csv.NewReader(fi).ReadAll()
	common.CheckError(err)
	return res
}

func SToI(s string) int {
	v, err := strconv.Atoi(s)
	common.CheckError(err)
	return v
}

func SToUI(s string) uint32 {
	v, err := strconv.ParseUint(s, 10, 32)
	common.CheckError(err)
	return uint32(v)
}

func (ipr *IPRegions) convertCSVToCountryInfo(data [][]string) (cis []CountryInfo) {
	for _, line := range data {
		ci := CountryInfo{}
		ci.Country = line[0]
		ci.A2 = line[1]
		ci.A3 = line[2]

		ci.Num = SToI(line[3])
		ci.DialingCode = line[4]
		cis = append(cis, ci)
	}
	return
}

func (ipr *IPRegions) convertCSVToCountryIPInfo(data [][]string) (ciis []CountryIPInfo) {
	for _, line := range data {
		ci := CountryIPInfo{}
		ci.IPFrom = SToUI(line[0])
		ci.IPTo = SToUI(line[1])
		ci.Country = line[6]
		ci.A2 = line[4]
		ci.A3 = line[5]
		ciis = append(ciis, ci)
	}
	return
}

func (ipr *IPRegions) InitFromFile(countryIPFile, countryInfoFile string) {
	countryInfos := ipr.convertCSVToCountryInfo(loadCSVFile(countryInfoFile))
	ipr.countryIPInfos = ipr.convertCSVToCountryIPInfo(loadCSVFile(countryIPFile))

	ipr.countryInfosAA = make(map[string]*CountryInfo)
	ipr.countryInfosNum = make(map[int]*CountryInfo)
	for i := range countryInfos {
		countryInfo := &countryInfos[i]
		ipr.countryInfosAA[ipr.getCountryInfoKey(countryInfo.A2, countryInfo.A3)] = countryInfo
		ipr.countryInfosNum[countryInfo.Num] = countryInfo
	}
}

func (ipr *IPRegions) getCountryInfoKey(a2, a3 string) string {
	return strings.ToLower(fmt.Sprintf("%s_%s", a2, a3))
}

func (ipr *IPRegions) getIPCountryInfo(sip string) *CountryInfo {
	ipinfo, ok := ipr.getIPCountryIPInfo(sip)
	if !ok {
		return nil
	}
	ctyInfo, ok := ipr.countryInfosAA[ipr.getCountryInfoKey(ipinfo.A2, ipinfo.A3)]
	if !ok {
		return nil
	}
	return ctyInfo
}

// GetIPCountryName ...
func (ipr *IPRegions) GetIPCountryName(sip string) string {
	if ctyInfo := ipr.getIPCountryInfo(sip); ctyInfo != nil {
		return ctyInfo.Country
	}
	return "unknown"
}

// GetCountryInfoByNum ...
func (ipr *IPRegions) GetCountryInfoByNum(countryNum int) (c *CountryInfo, b bool) {
	c, b = ipr.countryInfosNum[countryNum]
	return
}

func (ipr *IPRegions) getIPCountryIPInfo(sip string) (ctyInfo CountryIPInfo, ok bool) {
	ok = false
	ctinfos := ipr.countryIPInfos
	if ctinfos == nil {
		return
	}
	ctInfosCnt := len(ipr.countryIPInfos)
	ip := net.ParseIP(sip)
	if ip == nil {
		logger.Errorf("parse ip %s failed", sip)
		return

	}
	intip := ip2int(ip)

	n := sort.Search(ctInfosCnt, func(i int) bool {
		return ctinfos[i].IPFrom >= intip || ctinfos[i].IPTo >= intip
	})

	if n == ctInfosCnt || ctinfos[n].IPFrom > intip || ctinfos[n].IPTo < intip {
		//logger.Error("search ip " + sip + " country failed!")
	} else {
		ok = true
		ctyInfo = ctinfos[n]
	}
	return
}
