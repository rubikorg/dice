package postgres

import (
	"database/sql"
	"fmt"

	// this import is is used to only load the driver needed
	// if main package has it it will have to import all the
	// dialects driver
	_ "github.com/lib/pq"
	"github.com/rubikorg/dice"
)

// Connect to postgresql db using the connection uri
func Connect(uri dice.ConnectUri) (*sql.DB, error) {
	fmtStr := "host=%s port=%d user=%s password=%s dbname=%s "
	if !uri.SSL {
		fmtStr += "sslmode=disable"
	}

	cstr := fmt.Sprintf(fmtStr,
		uri.Host, uri.Port, uri.Username, uri.Password, uri.Database)
	return sql.Open("postgres", cstr)
}
