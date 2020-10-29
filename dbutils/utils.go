package dbutils

import "time"

// FmtLogTblName ...
func FmtLogTblName(tblName string, rectime time.Time) string {
	return tblName + "_" + rectime.Format("2006_01_02")
}
