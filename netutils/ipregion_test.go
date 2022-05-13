package netutils

import (
	"common"
	"context"
	"sort"
	"testing"

	"github.com/asmexie/go-logger/logger"
)

func TestSearchIP(t *testing.T) {
	InitTest(context.Background())
	defer UninitTest()
	ips := "114.114.114.114"
	logger.Debugf("got ip " + ips + " region at " + IPR().GetIPCountryName(ips))
	ips = "8.8.8.8"
	logger.Debugf("got ip " + ips + " region at " + IPR().GetIPCountryName(ips))

	ips = "77.88.99.11"
	logger.Debugf("got ip " + ips + " region at " + IPR().GetIPCountryName(ips))

	var countryInfos []CountryIPInfo
	var intip uint32
	intip = 16778241
	common.CheckError(GetDb().Select(&countryInfos,
		"select ip_from,ip_to,country from country_ip order by ip_from"))
	n := sort.Search(len(countryInfos), func(i int) bool {
		//logger.Debugf("find in i %v", i)
		return countryInfos[i].IPFrom >= intip || countryInfos[i].IPTo >= intip
	})

	if n == len(countryInfos) || countryInfos[n].IPFrom > intip || countryInfos[n].IPTo < intip {
		logger.Debugf("find failed n %v", n)
	} else {
		logger.Debugf("find in pos %d", n)
	}
}
