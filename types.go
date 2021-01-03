package dice

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v2"
)

type Seq struct {
	Order
	Key string
}

// Q is the query context for dice
type Q []primitive.E

// Comparison is the strings that represent comparison operators in SQL.
// The logical operators are also included in these constants.
type Comparison string

// LogicalComparison is the comparison to do logical operations
type LogicalComparison string

// Order strongly types asc and desc for you and used in
// SequenceStmt.
type Order string

// DriverIdent is the identification of a database driver for dice.
type DriverIdent string

// Model interface satisfies the the dice model requirements
// To become a dice model the caller must be able to
// convey their column-field bindings as ColumnLit, their
// primary key and the table that they satisfy.
type Model interface {
	// ColumnList returns the name of implementer's fields
	// in the order that their table is created.
	ColumnList() []string
	// PK is the definition of the table's primary key
	// and returns it's column name.
	PK() string
	// TableName returns the name of the table that the
	// implementer satisfies.
	TableName() string
	// Find let's you find the record that you seek by
	// accepting the dice.Q and a Seq specifying the order of the
	// result set.
	FindOne(Q, Seq)
}

// ConnectURI is the generalized struct for connecteing to your
// database drivers.
type ConnectURI struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Database string `yaml:"db" json:"db"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	SSL      bool   `yaml:"ssl" json:"ssl"`
}

// Structure is the definition of properties of a column. A dice
// field is nothing but the column of your table with it's
// structure definition provided inside `${tableName}.dice`.
type Structure struct {
	// The reason why these fields have json tags is because after
	// we get yaml.MapSlice we want to unmarshal column values
	// as dice.Structure
	Type          string `json:"type"`
	TablePK       bool   `json:"table_pk"`
	Unique        bool   `json:"unique"`
	AutoIncrement bool   `json:"auto_increment"`
	IsNotNull     bool   `json:"not_null"`
	Default       string `json:"default"`
	Constraint    string `json:"constraint"`
	Using         string `json:"using"`
	Through       string `json:"through"`
	Ignore        bool   `json:"ignore"`
	Reference     string `json:"ref"`
	// Mixins        []string `yaml:"mixins"`
}

// Schema defines your database table/collection. The schema definition
// is defined insde file.dice where file is your table name. The dice
// files is located inside source folder of your file system from which
// the target application compiles into the file.go dice Models.
type Schema struct {
	Table             string        `yaml:"table"`
	ModelName         string        `yaml:"model"`
	ShouldCreateDates bool          `yaml:"create_dates"`
	OrderedColumns    yaml.MapSlice `yaml:"columns"`
	ColumnAttrs       map[string]Structure
}

// The Options present in config.yaml
type Options struct {
	// Specifies for which dialect the models are being
	// generated. Without this config dice migrations will
	// not work.
	Dialect     DriverIdent `yaml:"connection"`
	Source      string      `yaml:"source"`
	Destination string      `yaml:"destination"`

	// Base defines what base statement for dice to generate
	Base string `yaml:"base"`
	// Filter is the implementation of type of filterStmt to generate
	// depending upon the dialect
	Filter string `yaml:"filter"`
	// Actions tells the compiler while running the
	// migrations you can look for additions in
	// columns or deletion of columns or not.
	// This will run CREATE COLUMN IF NOT EXISTS and
	// set it up automatically or deletes it if
	// cache has it and latest source does not.
	Actions struct {
		LookForAdditions bool `yaml:"no_additions"`
		LookForDeletions bool `yaml:"no_deletions"`
	} `yaml:"actions"`
	// Verbose tells compiler to log everything
	Verbose     bool       `yaml:"verbose"`
	Credentials ConnectURI `yaml:"credentials"`
}

// FieldData describes the conditions that must be satisfied for a
// dice.FilterStmt. The Condition field is the condition of the
// match if it is not mentioned it is assumed to be dice.Eq.
type FieldData struct {
	LogicalComparison
	Name      string
	Condition Comparison
	Value     interface{}
}

// columnDataHolder is the only builder in dice which holds the column
// action for Must() method
type columnDataHolder struct {
	column            string
	logicalComparison LogicalComparison
}

type modelData struct {
	BaseStmt  string
	Filter    string
	Dialect   DriverIdent
	ModelName string
	Columns   string
	PK        string
	TableName string
}

type colEquivalents struct {
	ColName string
	Kind    reflect.Kind
	Attr    Structure
}
