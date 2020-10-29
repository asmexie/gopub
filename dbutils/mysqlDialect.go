package dbutils

import (
	"fmt"
	"strings"
	"time"
)

const (
	timeFormat = "2006-01-02 15:04:05.000000"
)

// QuoteIdent ...
func QuoteIdent(s, quote string) string {
	part := strings.SplitN(s, ".", 2)
	if len(part) == 2 {
		return QuoteIdent(part[0], quote) + "." + QuoteIdent(part[1], quote)
	}
	return quote + s + quote
}

type mysqlDialect struct{}

func (d mysqlDialect) QuoteIdent(s string) string {
	return QuoteIdent(s, "`")
}

func (d mysqlDialect) EncodeString(s string) string {
	var buf strings.Builder

	buf.WriteRune('\'')
	// https://dev.mysql.com/doc/refman/5.7/en/string-literals.html
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case 0:
			buf.WriteString(`\0`)
		case '\'':
			buf.WriteString(`\'`)
		case '"':
			buf.WriteString(`\"`)
		case '\b':
			buf.WriteString(`\b`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case 26:
			buf.WriteString(`\Z`)
		case '\\':
			buf.WriteString(`\\`)
		default:
			buf.WriteByte(s[i])
		}
	}

	buf.WriteRune('\'')
	return buf.String()
}

func (d mysqlDialect) EncodeBool(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func (d mysqlDialect) EncodeTime(t time.Time) string {
	return `'` + t.Format(timeFormat) + `'`
}

func (d mysqlDialect) EncodeBytes(b []byte) string {
	return fmt.Sprintf(`0x%x`, b)
}

func (d mysqlDialect) Placeholder(_ int) string {
	return "?"
}
