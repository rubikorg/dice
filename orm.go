// Package dice provides you with simple and composable APIs for interacting
// with your database. It provides a model generator and DataStructure
// embeddinginterfaces.
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
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"reflect"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v2"
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

	Asc  Order = "asc"
	Desc       = "desc"

	Postgres DriverIdent = "postgres"
	MySQL                = "mysql"
	SQLite               = "sqlite"
	Mongo                = "mongo"
)

var orm = holder{}

type holder struct {
	db   *sql.DB
	mdb  *mongo.Database
	opts Options
}

func UseOpts(o Options) {
	orm.opts = o
}

func GetDiceOpts() (Options, error) {
	var opts Options
	b, err := ioutil.ReadFile("./dice.yaml")
	if err != nil {
		return opts, err
	}

	err = yaml.Unmarshal(b, &opts)
	if err != nil {
		return opts, err
	}

	return opts, nil
}

// Use sets driver identifier and the connection of
// database either sql.DB or mongo.Databse as the
// base consumer of running queries.
func Use(db interface{}, opts Options) {
	setLogger(true)

	if reflect.TypeOf(db).Elem() == reflect.TypeOf(sql.DB{}) &&
		isOneOf(string(opts.Dialect), "postgres", "sqlite", "mysql") {
		orm.db = db.(*sql.DB)
		log.Sugar().Debugf("Using sql.DB driver for %s", string(opts.Dialect))
	} else if reflect.TypeOf(db) == reflect.TypeOf(&mongo.Database{}) &&
		opts.Dialect == Mongo {
		orm.mdb = db.(*mongo.Database)
	} else {
		panic("dice: not a valid database connection object for driver name: " +
			string(opts.Dialect))
	}

	if orm.db == nil && orm.mdb == nil {
		panic("dice: no driver can be set")
	}

	orm.opts = opts
}

func GetDB() *mongo.Database {
	return orm.mdb
}

func isOneOf(val interface{}, rightOnes ...interface{}) bool {
	for _, ro := range rightOnes {
		if ro == val {
			return true
		}
	}

	return false
}

func PopulateAll(relation string, oids []primitive.ObjectID, target interface{}) error {
	// from relation map you can get the name of the struct
	// field to populate
	col := GetDB().Collection(relation)
	cursor, err := col.Aggregate(context.TODO(), Q{{"_id", bson.M{"$in": oids}}})
	if err != nil {
		return err
	}

	if err = cursor.All(context.TODO(), target); err != nil {
		return err
	}

	return nil
}

func Populate(relation string, oid primitive.ObjectID, target interface{}) error {
	col := GetDB().Collection(relation)
	res := col.FindOne(context.TODO(), Q{{"_id", oid}})
	if res == nil {
		return fmt.Errorf("cannot populate relation %s", relation)
	}

	if err := res.Decode(target); err != nil {
		return err
	}

	return nil
}
