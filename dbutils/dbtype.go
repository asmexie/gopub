package dbutils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/muroachanf/dbr"

	"github.com/asmexie/gopub/common"

	"github.com/go-sql-driver/mysql"
)

// NullInt64 is an alias for sql.NullInt64 data type
type NullInt64 sql.NullInt64

// NullBool is an alias for sql.NullBool data type
type NullBool sql.NullBool

// NullFloat64 is an alias for sql.NullFloat64 data type
type NullFloat64 sql.NullFloat64

// NullString is an alias for sql.NullString data type
type NullString sql.NullString

// NullTime is an alias for mysql.NullTime data type
type NullTime mysql.NullTime

// CUID ...
type CUID int64

var gCUIDDevID int64
var gCUIDBuilder = make(chan CUID)

// MarshalJSON ...
func (cuid CUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(cuid), 10))
}

// UnmarshalJSON ...
func (cuid *CUID) UnmarshalJSON(v []byte) error {
	var s string
	err := json.Unmarshal(v, &s)
	if err != nil {
		return err
	}
	c, err := strconv.ParseInt(s, 10, 64)
	if err == nil {
		*cuid = CUID(c)
	}

	return err
}

func (cuid CUID) String() string {
	return strconv.FormatInt(int64(cuid), 10)
}

// StrToCUID ...
func StrToCUID(s string) CUID {
	if s == "" {
		return 0
	}
	return CUID(common.StrToInt64(s, 0))
}

// InitCuidDev ...
func InitCuidDev(devid int64) {
	if gCUIDDevID != 0 {
		return
	}
	gCUIDDevID = devid
	startTime := time.Date(2018, 1, 1, 0, 0, 0, 0, time.Local)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				common.LogError(e)
			}
		}()
		var lastTime int64
		var counter, lastCounter int64

		var timeMax int64 = 0x1FFFFFFFF
		var counterMax int64 = 0xFFFF
		var devIDMax int64 = 0x3FFF
		if gCUIDDevID > devIDMax {
			panic(fmt.Errorf(" got invalid cuid device id when generate cuid"))
		}

		for {

			now := int64(time.Now().Sub(startTime).Seconds())

			if now < 0 || now >= timeMax {
				panic(fmt.Errorf(" got invalid time when generate cuid"))
			}
			counter++
			if lastTime != now {
				lastCounter = counter
				lastTime = now
			} else {
				if counter >= counterMax {
					counter = 1
				}
				if counter == lastCounter {
					time.Sleep(time.Microsecond)
					continue
				}
			}

			// | 33 位时间 | 16位 counter| 14 位设备id|->共计63位数，防止出现负数
			id := (now << 30) | (counter << 14) | gCUIDDevID
			gCUIDBuilder <- CUID(id)
		}
	}()
}

// NewCUID ...
func NewCUID() CUID {
	if gCUIDDevID == 0 {
		panic(errors.New("not init device id when new cuid"))
	}
	return <-gCUIDBuilder
}

// NewDBTime ...
func NewDBTime(tm time.Time) DBTime {
	return DBTime{Time: tm}
}

// DBTime ...
type DBTime struct {
	time.Time
}

// UnmarshalXML ...
func (tm *DBTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	*tm = DBTime{Time: common.StrToTime(s)}
	return nil
}

// MarshalXML ...
func (tm DBTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(common.TimeToStr(tm.Time), start)
}

// Value ...
func (tm DBTime) Value() (driver.Value, error) {
	return tm.Time, nil
}

// String ...
func (tm DBTime) String() string {
	return common.TimeToStr(tm.Time)
}

// Scan ...
func (tm *DBTime) Scan(value interface{}) error {
	if value == nil {
		// set the value of the pointer yne to YesNoEnum(false)
		return dbr.ErrNotSupported
	}

	t, ok := value.(time.Time)
	if ok {
		*tm = DBTime{Time: t}
		return nil
	}
	v, ok := value.([]uint8)
	if !ok {
		return fmt.Errorf("not support time type %v", reflect.TypeOf(v))
	}

	*tm = DBTime{Time: common.StrToTime(string(v))}

	return nil
}

// MarshalJSON ...
func (tm DBTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(common.TimeToStr(tm.Time))
}

// UnmarshalJSON ...
func (tm *DBTime) UnmarshalJSON(v []byte) error {
	var s string
	err := json.Unmarshal(v, &s)
	if err != nil {
		return err
	}
	newtm, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		newtm, err = time.ParseInLocation("2006-01-02", s, time.Local)
	}

	if err == nil {
		*tm = DBTime{Time: newtm}
	}
	return err
}

// DBIntBool ...
type DBIntBool int

// Value ...
func (bv DBIntBool) Value() bool {
	return bv != 0
}

// MarshalJSON custom marshal json
func (bv DBIntBool) MarshalJSON() ([]byte, error) {
	return json.Marshal(bv.Value())
}

// UnmarshalJSON for custom unmarshal json
func (bv *DBIntBool) UnmarshalJSON(b []byte) error {
	var value bool
	err := json.Unmarshal(b, &value)
	if err != nil {
		return err
	}
	if value {
		*bv = 1
	} else {
		*bv = 0
	}
	return nil
}

// DBBool used for bool type (bit(1)) in mysql
type DBBool []uint8

// Value ...
func (bv DBBool) Value() bool {
	return len(bv) > 0 && bv[0] != 0
}

// UnmarshalJSON for custom unmarshal json
func (bv DBBool) UnmarshalJSON(b []byte) error {
	var value bool
	err := json.Unmarshal(b, &value)
	if err != nil {
		return err
	}
	v := uint8(0)
	if value {
		v = 1
	}
	if len(bv) == 0 {
		bv = append(bv, v)
	} else {
		bv[0] = v
	}
	return nil
}

// MarshalJSON custom marshal json
func (bv DBBool) MarshalJSON() ([]byte, error) {
	value := false
	if len(bv) > 0 && bv[0] != 0 {
		value = true
	}
	return json.Marshal(value)
}

// NewDBDate ...
func NewDBDate(tm time.Time) DBDate {
	return DBDate{Time: dateFromTime(tm)}
}

// DBDate ...
type DBDate struct {
	time.Time
}

func dateFromTime(tm time.Time) time.Time {
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
}

// UnmarshalXML ...
func (tm *DBDate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	*tm = DBDate{Time: dateFromTime(common.StrToTime(s))}
	return nil
}

// MarshalXML ...
func (tm DBDate) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(common.DateToStr(tm.Time), start)
}

// Value ...
func (tm DBDate) Value() (driver.Value, error) {
	return tm.Time, nil
}

// String ...
func (tm DBDate) String() string {
	return common.DateToStr(tm.Time)
}

// Scan ...
func (tm *DBDate) Scan(value interface{}) error {
	if value == nil {
		// set the value of the pointer yne to YesNoEnum(false)
		return dbr.ErrNotSupported
	}

	t, ok := value.(time.Time)
	if ok {
		*tm = DBDate{Time: t}
		return nil
	}
	v, ok := value.([]uint8)
	if !ok {
		return fmt.Errorf("not support time type %v", reflect.TypeOf(v))
	}

	*tm = DBDate{Time: dateFromTime(common.StrToTime(string(v)))}

	return nil
}

// MarshalJSON ...
func (tm DBDate) MarshalJSON() ([]byte, error) {
	return json.Marshal(common.DateToStr(tm.Time))
}

// UnmarshalJSON ...
func (tm *DBDate) UnmarshalJSON(v []byte) error {
	var s string
	err := json.Unmarshal(v, &s)
	if err != nil {
		return err
	}
	newtm, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		newtm, err = time.ParseInLocation("2006-01-02", s, time.Local)
	}

	if err == nil {
		*tm = DBDate{Time: dateFromTime(newtm)}
	}
	return err
}
