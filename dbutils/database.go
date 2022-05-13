package dbutils

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/asmexie/gopub/common"
	"github.com/muroachanf/dbr"
	"github.com/muroachanf/dbr/dialect"
	"github.com/muroachanf/go-logger/logger"
)

// Database ...
type Database struct {
	*dbr.Connection
	ctx     context.Context
	evtr    dbr.EventReceiver
	connStr string
	dbf     *DBConfig
}

// Close ...
func (dbs *Database) Close() {
	if dbs.Connection != nil {
		dbs.Connection.Close()
		dbs.Connection = nil
	}

}

// NewDatabase ...
func NewDatabase(ctx context.Context, evtr dbr.EventReceiver) *Database {
	if evtr == nil {
		evtr = &DBEventReceiver{}
	}
	return &Database{ctx: ctx, evtr: evtr}
}

// Context ...
func (dbs *Database) Context() context.Context {
	return dbs.ctx
}

func dbrOpen(driver, dsn string, log dbr.EventReceiver) (*dbr.Connection, error) {
	if log == nil {
		log = &dbr.NullEventReceiver{}
	}
	logger.Debugf("start connect %v server:%v", driver, dsn)
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	var d dbr.Dialect
	switch driver {
	case "mysql":
		d = mysqlDialect{}
	case "postgres":
		d = dialect.PostgreSQL
	case "sqlite3":
		d = dialect.SQLite3
	default:
		return nil, dbr.ErrNotSupported
	}
	return &dbr.Connection{DB: conn, EventReceiver: log, Dialect: d}, nil
}

// InitDBF ...
func (dbs *Database) InitDBF(tblName, keyCol, valueCol, enableCol string) *DBConfig {
	dbs.dbf = NewDBConfig(tblName, keyCol, valueCol, enableCol)

	return dbs.dbf
}

// StartUpdateDBF ...
func (dbs *Database) StartUpdateDBF(interval time.Duration) {
	go common.TimerAction(dbs.ctx, interval, "update-dbf", func() {
		common.LogError(dbs.UpdateDBF())
	})
}

// DBF ...
func (dbs *Database) DBF() *DBConfig {
	return dbs.dbf
}

// UpdateDBF ...
func (dbs *Database) UpdateDBF() error {
	return dbs.dbf.Load(dbs.NewSession())
}

// Open ...
func (dbs *Database) Open(connStr string) error {
	logger.Debugf("open db:%v", connStr)
	defer logger.Debugf("open db over:%v", connStr)
	dbs.Close()
	for {

		args := strings.Split(connStr, ",")
		if len(args) < 2 {
			panic(fmt.Errorf("arg  <<%s>> size less than %d", strings.Join(args, " "), 2))
		}

		if len(args) >= 3 {
			devID, err := strconv.ParseInt(args[2], 10, 64)
			common.CheckError(err)
			InitCuidDev(devID)
		}

		db, err := dbrOpen(args[0], args[1], dbs.evtr)
		if err == nil {
			dbs.Connection = db
			return nil
		}
		dbs.Connection = db

		common.LogError(err)
		select {
		case <-time.After(5 * time.Second):
			logger.Debugf("reconnect db")
		case <-dbs.ctx.Done():
			logger.Error(dbs.ctx.Err())
			return dbs.ctx.Err()
		}
	}
}

// NewSession ...
func (dbs *Database) NewSession() *DBSession {
	return &DBSession{
		Session: dbs.Connection.NewSession(dbs.evtr),
		ctx:     dbs.ctx,
		db:      dbs,
	}
}

// DBExecer ...
type DBExecer interface {
	Exec() (sql.Result, error)
}

// CheckExec ...
func CheckExec(execer DBExecer) bool {
	r, err := execer.Exec()
	if err != nil {
		common.LogError(err)
		return false
	}
	n, err := r.RowsAffected()
	if err != nil {
		common.LogError(err)
		return false
	}
	return n > 0
}

// DBEventReceiver ...
type DBEventReceiver struct {
	*dbr.NullEventReceiver
}

// TimingKv ...
func (er *DBEventReceiver) TimingKv(eventName string, nanoseconds int64, kvs map[string]string) {
	seconds := time.Duration(nanoseconds).Seconds()
	if int(seconds) > 3 {
		logger.Debugf("%v too slow time %v, sql:%v", eventName, seconds, kvs)
	}
}
