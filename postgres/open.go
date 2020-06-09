package postgres

import (
	"database/sql"
	"fmt"

	// this import is is used to only load the driver needed
	// if main package has it it will have side effects of
	// all the dialects driver
	_ "github.com/lib/pq"
	"github.com/rubikorg/dice"
)

// Connect to postgresql db using the connection uri
func Connect(uri dice.ConnectUri) (*sql.DB, error) {
	var connStr = ""
	connStr += "host=" + uri.Host
	connStr += fmt.Sprintf(" port=%d", uri.Port)
	connStr += " user=" + uri.Username
	connStr += " dbname=" + uri.Database

	if !uri.SSL {
		connStr += " sslmode=disable"
	}

	if uri.Password != "" {
		connStr += " password=" + uri.Password
	}

	return sql.Open(string(dice.Postgres), connStr)
}
