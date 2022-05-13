package dbutils

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/asmexie/dbr"
	"github.com/asmexie/go-logger/logger"
)

// DBSession ...
type DBSession struct {
	*dbr.Session
	ctx context.Context
	db  *Database
}

// ExecBuilderAndCommit ...
func (ds DBSession) ExecBuilderAndCommit(tx *dbr.Tx, execer DBExecer) bool {
	if CheckExec(execer) {
		tx.Commit()

		return true
	}
	return false
}

// VerifyInsertExec ...
func (ds *DBSession) VerifyInsertExec(affectRows int64, st *dbr.InsertStmt) error {
	rs, err := st.Exec()
	if err != nil {
		return err
	}
	n, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	if n != affectRows {
		return fmt.Errorf("insert affect rows is %v, expect %v", n, affectRows)
	}
	return nil
}

// QDBColumns ...
func (ds *DBSession) QDBColumns(data interface{}, quote bool, discardCols ...string) DBColumns {
	dataType := reflect.TypeOf(data)
	colums := []string{}
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
	}
	if dataType.Kind() != reflect.Struct {
		panic("only support struct type")
	}
	for i := 0; i < dataType.NumField(); i++ {
		tag := dataType.Field(i).Tag.Get("db")
		if tag == "-" {
			// ignore
			continue
		}
		if tag == "" {
			// no tag, but we can record the field name
			continue
		}
		if common.SSContains(discardCols, tag) {
			continue
		}
		if quote {
			tag = ds.QuoteIdent(tag)
		}
		colums = append(colums, tag)
	}
	return colums
}

// DBColumns ...
type DBColumns []string

// Prefix ...
func (cols DBColumns) Prefix(prefix string) DBColumns {
	for i := range cols {
		cols[i] = prefix + "." + cols[i]
	}
	return cols
}

// DBSelColumns ...
func (ds *DBSession) DBSelColumns(data interface{}, discardCols ...string) DBColumns {
	return ds.QDBColumns(data, true, discardCols...)
}

// DBInsColumns ...
func (ds *DBSession) DBInsColumns(data interface{}, discardCols ...string) DBColumns {
	return ds.QDBColumns(data, false, discardCols...)
}

// BeginTx ...
func (ds *DBSession) BeginTx() (*dbr.Tx, error) {
	return ds.Session.BeginTx(ds.db.Context(), nil)
}

// DBUpdateMap ...
func (ds *DBSession) DBUpdateMap(data interface{}, discardCols ...string) map[string]interface{} {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)

	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}
	if dataType.Kind() != reflect.Struct {
		panic("only support struct type")
	}
	m := map[string]interface{}{}
	for i := 0; i < dataType.NumField(); i++ {
		field := dataType.Field(i)
		tag := field.Tag.Get("db")
		if tag == "-" {
			// ignore
			continue
		}
		if tag == "" {
			// no tag, but we can record the field name
			continue
		}
		if common.SSContains(discardCols, tag) {
			continue
		}

		m[tag] = dataValue.Field(i).Interface()
	}
	return m
}

// Context ...
func (ds DBSession) Context() context.Context {
	return ds.ctx
}

// CheckTableExists ...
func (ds DBSession) CheckTableExists(tableName string) bool {
	sql := `SELECT COUNT(1) 
    	FROM information_schema.TABLES as info
    	where table_schema=database() and table_name=?`
	var count int
	common.CheckError(ds.SelectBySql(sql, tableName).LoadOne(&count))
	return count > 0
}

var autoIncSQLRegExp = regexp.MustCompile("(?i)AUTO_INCREMENT=\\d+")

// CheckDropTable ...
func (ds DBSession) CheckDropTable(tableName string, date time.Time) {
	tblName := fmt.Sprintf("%s_%s", tableName, date.Format("2006_01_02"))
	if ds.CheckTableExists(tblName) {
		sql := fmt.Sprintf("drop table `%s`", tblName)
		logger.Debugf("exec %s", sql)
		if _, err := ds.Exec(sql); err != nil {
			common.LogError(err)
		}
	}
}

// CheckLogTables ...
func (ds DBSession) CheckLogTables(tableNames []string, remainDay int) {
	for _, tableName := range tableNames {
		ds.CheckLogTable(tableName, remainDay)
	}
}

// RecordToMap ...
func (ds DBSession) RecordToMap(data interface{}, discardCols ...string) common.Map {
	dataType := reflect.TypeOf(data)
	dataValue := reflect.ValueOf(data)
	if dataType.Kind() == reflect.Ptr {
		dataType = dataType.Elem()
		dataValue = dataValue.Elem()
	}
	m := common.Map{}
	if dataType.Kind() != reflect.Struct {
		panic("only support struct type")
	}
	for i := 0; i < dataType.NumField(); i++ {
		tag := dataType.Field(i).Tag.Get("db")
		if tag == "-" {
			// ignore
			continue
		}
		if tag == "" {
			// no tag, but we can record the field name
			continue
		}
		if common.SSContains(discardCols, tag) {
			continue
		}

		fdValue := dataValue.Field(i)
		m[tag] = fdValue.Interface()
	}
	return m
}

// CheckLogTable ...
func (ds DBSession) CheckLogTable(tableName string, remainDay int) {

	logger.Debugf("checking log table %s", tableName)

	type CreateSQL struct {
		TableName string `db:"Table"`
		CreateSQL string `db:"Create Table"`
	}
	var createSQL CreateSQL
	//var createSql string
	common.CheckError(ds.SelectBySql(fmt.Sprintf("show create table `%s`", tableName)).LoadOne(&createSQL))

	sql := autoIncSQLRegExp.ReplaceAllString(createSQL.CreateSQL, "")
	re := regexp.MustCompile(`(?i)` + tableName)

	now := time.Now()

	tblName := fmt.Sprintf("%s_%s", tableName,
		now.Add(time.Hour*24).Format("2006_01_02"))
	logger.Debugf("checking log table %s", tblName)

	tblName = fmt.Sprintf("%s_%s", tableName, now.Format("2006_01_02"))
	if !ds.CheckTableExists(tblName) {
		ds.Exec(re.ReplaceAllString(sql, tblName))
	}

	tblName = fmt.Sprintf("%s_%s", tableName,
		now.Add(time.Hour*24).Format("2006_01_02"))
	if !ds.CheckTableExists(tblName) {
		ds.Exec(re.ReplaceAllString(sql, tblName))
	}

	remainLogDays := 7
	for i := 0; i < 90; i++ {
		ds.CheckDropTable(tableName,
			now.Add(-1*time.Duration(remainLogDays+1+i)*time.Hour*24))
	}
}

// InsertInto ...
func (ds DBSession) InsertInto(table string) *InsertStmt {
	return &InsertStmt{
		InsertStmt: ds.Session.InsertInto(table),
	}
}

// InsertStmt ...
type InsertStmt struct {
	*dbr.InsertStmt
}
