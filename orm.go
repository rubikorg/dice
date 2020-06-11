// Package dice provides you with simple and composable APIs for interacting with your
// database. It provides a model generator and Object-Relational-Mapping interfaces.
//
// Initializing dice:
//
// 		db, _ := postgres.Connect(uri) // dice.ConnectUri
// 		dice.Use(dice.Postgres, db) // driver identifier, connection object
//
// A simple query to find record by primary-key:
//
//		var g GameModel
//		f := dice.Filter()
// 		f.ById(1)
//		err := models.Game(&g).Find(f, nil)
// 		fmt.Println(g) // view your data
package dice

import (
	"database/sql"
	"reflect"

	"go.mongodb.org/mongo-driver/mongo"
)

const (
	Eq    Comparison = " = "
	Neq              = " != "
	Gt               = " > "
	Lt               = " < "
	Btwn             = " BETWEEN "
	Nbtwn            = " NOT BETWEEN "
	In               = " IN "
	Nin              = " NOT IN "
	Like             = " LIKE "
	Nlike            = " NOT LIKE "

	OR  LogicalComparison = " OR "
	AND                   = " AND "

	Asc  SequenceOpts = "asc"
	Desc              = "desc"

	Postgres DriverIdent = "postgres"
	MySQL                = "mysql"
	SQLite               = "sqlite"
	Mongo                = "mongo"
)

var orm = holder{}

type holder struct {
	compilerCache
	db     *sql.DB
	mdb    *mongo.Database
	driver DriverIdent
}

// Use sets driver identifier and the connection of
// database either sql.DB or mongo.Databse as the
// base consumer of running queries.
func Use(driver DriverIdent, db interface{}, opts ...Options) {
	if len(opts) > 0 && opts[0].Verbose {
		setLogger(true)
	} else {
		setLogger(false)
	}

	p := getCachePath()
	err := decodeCompilerCache(p, &orm.compilerCache)
	if err != nil {
		panic(err)
	}

	if reflect.TypeOf(db).Elem() == reflect.TypeOf(sql.DB{}) &&
		isOneOf(string(driver), "postgres", "sqlite", "mysql") {
		orm.db = db.(*sql.DB)
		// slog.Info("Using sql.DB driver for ", string(driver))
	} else if reflect.TypeOf(db).Elem() == reflect.TypeOf(mongo.Database{}) &&
		driver == Mongo {
		orm.mdb = db.(*mongo.Database)
	} else {
		panic("dice: not a valid database connection object for driver name: " + string(driver))
	}

	orm.driver = driver
}

// Seq returns a empty map of dice.ResultSequence which satisfies
// the SequenceStmt interface. This is mosttly used inside
// Base.Find(filter, Seq()) to order your result
func Seq() SequenceStmt {
	return make(ResultSequence)
}

// Single returns a FilterStmt depending upon the dialect
// of your SQL database / NoSQL database
func Single(column string, value interface{}) FilterStmt {
	switch orm.driver {
	case Postgres, MySQL, SQLite:
		qf := SQLFilter{}
		if column != "" {
			cv := FieldData{
				Name:  column,
				Value: value,
			}
			qf.columnValues = append(qf.columnValues, cv)
		}

		qf.limit = 1
		return &qf
	default:
		return &SQLFilter{}
	}
}

func isOneOf(val string, rightOnes ...string) bool {
	for _, ro := range rightOnes {
		if ro == val {
			return true
		}
	}

	return false
}
