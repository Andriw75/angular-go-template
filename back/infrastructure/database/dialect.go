package database

import (
	"strconv"
	"strings"
)

type Dialect interface {
	Name() string
	Rebind(query string) string
	AutoIncrementPK() string
	FloatType() string
}

type postgresDialect struct{}

func (postgresDialect) Name() string { return "postgres" }

func (postgresDialect) Rebind(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteByte(query[i])
	}
	return b.String()
}

func (postgresDialect) AutoIncrementPK() string { return "BIGSERIAL PRIMARY KEY" }
func (postgresDialect) FloatType() string       { return "DOUBLE PRECISION" }

type sqliteDialect struct{}

func (sqliteDialect) Name() string { return "sqlite" }

func (sqliteDialect) Rebind(query string) string { return query }

func (sqliteDialect) AutoIncrementPK() string { return "INTEGER PRIMARY KEY AUTOINCREMENT" }
func (sqliteDialect) FloatType() string       { return "REAL" }
