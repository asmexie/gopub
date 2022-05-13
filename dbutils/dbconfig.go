package dbutils

import (
	"strconv"

	"github.com/asmexie/gopub/common"
)

// DBConfig ...
type DBConfig struct {
	cc        map[string]string
	keyCol    string
	valueCol  string
	enableCol string
	tblName   string
}

// NewDBConfig ...
func NewDBConfig(tblName, keyCol, valueCol, enableCol string) *DBConfig {
	df := DBConfig{
		cc:        map[string]string{},
		keyCol:    keyCol,
		valueCol:  valueCol,
		enableCol: enableCol,
		tblName:   tblName,
	}
	return &df
}

// Load ...
func (df *DBConfig) Load(ds *DBSession) error {
	sel := ds.Select(df.keyCol, df.valueCol).From(df.tblName)
	if df.enableCol != "" {
		sel = sel.Where(df.enableCol + "<> 0")
	}

	rows, err := sel.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	newCC := map[string]string{}
	for rows.Next() {
		var keyName, valueName string
		err = rows.Scan(&keyName, &valueName)
		if err != nil {
			return err
		}
		newCC[keyName] = valueName
	}
	df.cc = newCC

	return nil
}

// IntValue ...
func (df DBConfig) IntValue(key string) (int, bool) {

	v, ok := df.cc[key]
	if !ok {
		return 0, false
	}

	value, err := strconv.Atoi(v)
	common.CheckError(err)

	return value, true
}

// IntValueDef ...
func (df DBConfig) IntValueDef(key string, defValue int) int {
	v, ok := df.IntValue(key)
	if !ok {
		return defValue
	}
	return v
}

// StrValueDef ...
func (df DBConfig) StrValueDef(key, defValue string) string {
	v, ok := df.StrValue(key)
	if !ok {
		return defValue
	}
	return v
}

// StrValue ...
func (df DBConfig) StrValue(key string) (string, bool) {

	v, ok := df.cc[key]
	if !ok {
		return "", false
	}
	return v, true
}
