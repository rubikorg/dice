package dice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Comparison is the strings that represent comparison operators in SQL.
// The logical operators are also included in these constants.
type Comparison string

type LogicalComparison string

// SequenceOpts strongly types asc and desc for you and used in
// SequenceStmt.
type SequenceOpts string

// DriverIdent is the identification of a database driver for dice.
type DriverIdent string

// PqBase is the implementation of dice.BaseStmt for PostgreSQL
type PqBase struct {
	target   interface{}
	ctx      context.Context
	cancel   context.CancelFunc
	table    string
	filter   FilterStmt
	seq      SequenceStmt
	query    string
	values   []interface{}
	isSingle bool
}

type ResultSequence map[string]SequenceOpts

// SQLFilter satisfies dice.FilterStmnt for SQL database drivers
// it uses sql.DB connection to query and currently supports only
// PostgreSQL driver. Other drivers can be used with the driverIdent
// passed in the Use() method but it is not tested.
type SQLFilter struct {
	limit        int
	offset       int
	columnValues []FieldData
	err          error
	selection    []string
}

// columnDataHolder is the only builder in dice which holds the column
// action for Must() method
type columnDataHolder struct {
	qf                *SQLFilter
	column            string
	logicalComparison LogicalComparison
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

// Target implements dice.BaseStmt.
func (pb PqBase) Target(t interface{}, ctx ...context.Context) BaseStmt {
	_, err := checkTarget(t)
	if err != nil {
		panic(err)
	}

	base := PqBase{target: t}
	if len(ctx) > 0 {
		// TODO: change this part to get context from filter
		// may be filter must have a func called WithContext????
		// idk!!
		base.ctx = ctx[0]
	} else {
		c, cf := context.WithCancel(context.TODO())
		base.ctx = c
		base.cancel = cf
	}

	return base
}

// Find implements dice.BaseStmt for SELECT clause in SQL.
func (pb PqBase) Find(f FilterStmt, seq SequenceStmt) error {
	pb.filter = f
	pb.seq = seq

	model := createTargetModel(pb.target)

	err := generateSelectQuery(&pb, model)
	if err != nil {
		return err
	}

	log.Debug("Executing find method",
		zap.String("query", pb.query),
		zap.Object("filter", sqlFilterMarshaler{f.(*SQLFilter)}),
	)

	if f != nil {
		qf := f.(*SQLFilter)
		if qf.limit == 1 {
			return querySingle(&pb, model)
		}

		// if targetElem.Kind() != reflect.Array &&
		// 	targetElem.Kind() != reflect.Slice {
		// 	msg := "this query can return multiple rows. please use []%s"
		// 	return fmt.Errorf(msg, reflect.TypeOf(b.target).Elem().Name())
		// }
		return queryMultiple(&pb, model, pb.target)
	}

	return queryMultiple(&pb, model, pb.target)
}

// checkTarget verifies if the target provided by the Base.Target()
// is an implementation of dice.Model or bit
func checkTarget(target interface{}) (Model, error) {
	targetElem := reflect.TypeOf(target).Elem()
	singularTarget := reflect.New(targetElem).Interface()
	if targetElem.Kind() == reflect.Slice ||
		targetElem.Kind() == reflect.Array {
		// because it is a slice ..we dereference and get the underlying type
		// by calling Elem twice
		singularTarget = reflect.New(targetElem.Elem()).Interface()
	}

	modelType := reflect.TypeOf((*Model)(nil)).Elem()
	if !reflect.ValueOf(singularTarget).Type().Implements(modelType) {
		return nil, fmt.Errorf("target struct %s is not a dice.Model",
			reflect.TypeOf(singularTarget))
	}

	return createTargetModel(target), nil
}

func querySingle(b *PqBase, m Model) error {
	row := orm.db.QueryRowContext(b.ctx, b.query, b.values...)
	return scanSingle(m, b.target, row)
}

func scanSingle(model Model, target interface{}, row *sql.Row) error {
	var columns []interface{}
	for _, c := range model.ColumnList() {
		value := reflect.ValueOf(target).Elem().FieldByName(c)
		columns = append(columns, value.Addr().Interface())
	}

	err := row.Scan(columns...)
	if err != nil {
		return err
	}

	return nil
}

func queryMultiple(b *PqBase, model Model, target interface{}) error {
	rows, err := orm.db.QueryContext(b.ctx, b.query, b.values...)
	if err != nil {
		return err
	}

	return scanMultiple(b, model, rows, target)
}

func scanMultiple(b *PqBase, model Model, rows *sql.Rows, target interface{}) error {
	defer rows.Close()

	targetSliceVal := reflect.ValueOf(target).Elem()
	structType := targetSliceVal.Type().Elem()
	targetStructVal := reflect.New(structType).Elem()

	// collect values for scan
	var values []interface{}
	slt := b.filter.(*SQLFilter).selection

	for _, e := range model.ColumnList() {
		if len(slt) > 0 && isOneOf(strings.ToLower(e), slt...) {
			values = append(values, targetStructVal.FieldByName(e).Addr().Interface())
		} else {
			values = append(values, targetStructVal.FieldByName(e).Addr().Interface())
		}
	}

	for rows.Next() {
		err := rows.Scan(values...)
		if err != nil {
			return err
		}

		targetSliceVal.Set(reflect.Append(targetSliceVal, targetStructVal))
	}

	return nil
}

// createTargetModel creates a new instance of target and returns it as
// a dice.Model
func createTargetModel(target interface{}) Model {
	targetElem := reflect.TypeOf(target).Elem()
	var structType reflect.Type
	if targetElem.Kind() == reflect.Slice ||
		targetElem.Kind() == reflect.Array {
		targetSliceVal := reflect.ValueOf(target).Elem()
		structType = targetSliceVal.Type().Elem()
		return reflect.New(structType).Elem().Interface().(Model)
	}

	return reflect.New(targetElem).Interface().(Model)
}

// Update implements dice.BaseStmt.
func (PqBase) Update(f FilterStmt) error {
	return nil
}

// Delete implements dice.BaseStmt.
func (pb PqBase) Delete(filter FilterStmt) error {
	model := createTargetModel(pb.target)

	pb.filter = filter
	if filter == nil {
		return errors.New("Delete() requires a filter. If you want to delete every " +
			"record, use DeleteAll() method.")
	}

	q := fmt.Sprintf("DELETE FROM \"%s\" WHERE ", model.TableName())
	whereClause := (&pb).getSQLWhere()
	q += whereClause + ";"

	_, err := orm.db.ExecContext(pb.ctx, q, pb.values...)
	if err != nil {
		return err
	}

	return nil
}

// Create implements dice.BaseStmt for SQL
func (pb PqBase) Create() (Result, error) {
	model := createTargetModel(pb.target)
	var query string
	var values []interface{}
	var pkVal interface{}
	if pb.target == nil {
		query = fmt.Sprintf("INSERT INTO \"%s\" DEFAULT VALUES", model.TableName())
	} else {
		cols := orm.compilerCache.ColumnAttrs[model.TableName()]
		tval := reflect.ValueOf(pb.target).Elem()
		colVals := []string{}
		valTempl := []string{}
		vCount := 1
		for _, c := range cols {
			key := fmt.Sprintf("%s.%s", model.TableName(), c)
			fieldName := orm.compilerCache.ColEquivalents[key]
			if c != model.PK() {
				tvalField := tval.FieldByName(fieldName.ColName).Interface()
				switch tvalField.(type) {
				case sql.NullTime:
					t := tvalField.(sql.NullTime)
					if t.Time.IsZero() {
						t.Time = time.Now()
					}

					values = append(values, t.Time.Format(time.RFC3339))
					break
				case int:
					values = append(values, tvalField)
					break
				default:
					values = append(values, tvalField)
					break
				}

				colVals = append(colVals, c)
				valTempl = append(valTempl, fmt.Sprintf("$%d", vCount))
				vCount++
			} else {
				pkVal = tval.FieldByName(fieldName.ColName).Addr().Interface()
			}
		}

		createTempl := "INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING %s;"
		query = fmt.Sprintf(createTempl, model.TableName(),
			strings.Join(colVals, ", "), strings.Join(valTempl, ", "), model.PK())

		fmt.Println("query:", query)
	}

	err := orm.db.QueryRowContext(pb.ctx, query, values...).Scan(pkVal)
	return nil, err
}

// Cancel lets you cancel the execution of current query
func (pq PqBase) Cancel() {
	pq.cancel()
}

// Must defines a condition over a column which adds it in WHERE clause
func (cdh *columnDataHolder) Must(cond Comparison, value interface{}) {
	cdh.qf.addConditions(cdh.column, cdh.logicalComparison, cond, value)
}

// Chunk returns a chunk of recordset of your query. Look
// for dice.FilterStmnt for more details.
func (qf *SQLFilter) Chunk(limit, offset int) {
	qf.limit = limit
	qf.offset = offset
}

// Or puts together a logical or clause for your WHERE statement.
// This is a custom implementation only for dice.SQLFilter
func (qf *SQLFilter) Or(column string) *columnDataHolder {
	cd := columnDataHolder{
		qf:                qf,
		column:            column,
		logicalComparison: OR,
	}
	return &cd
}

func (qf *SQLFilter) addConditions(
	column string, lcomp LogicalComparison, cond Comparison, value interface{}) {

	if reflect.TypeOf(value).Kind() != reflect.Func ||
		reflect.TypeOf(value).Kind() != reflect.Chan ||
		reflect.TypeOf(value).Kind() != reflect.Struct {

		validCond := cond
		s := string(cond)
		if s[0] != ' ' || s[len(cond)-1] != ' ' {
			s = strings.TrimSpace(s)
			s = fmt.Sprintf(" %s ", s)
			validCond = Comparison(s)
		}

		cd := FieldData{
			Name:              column,
			Condition:         validCond,
			LogicalComparison: lcomp,
			Value:             value,
		}
		qf.columnValues = append(qf.columnValues, cd)
	}
}

// Field is dice.FilterStmt method.
func (qf *SQLFilter) Field(name string) *columnDataHolder {
	return &columnDataHolder{qf: qf, logicalComparison: AND, column: name}
}

// Match implemented for SQLFilter from FilterStmt.
func (qf *SQLFilter) Match(data []FieldData) {
	for _, d := range data {
		if d.LogicalComparison == "" {
			d.LogicalComparison = AND
		}

		if d.Condition == "" {
			d.Condition = Eq
		}

		qf.addConditions(d.Name, d.LogicalComparison, d.Condition, d.Value)
	}
}

// Pick implements selection for SQL.
func (qf *SQLFilter) Pick(fields ...string) {
	qf.selection = fields
}

// Asc implements SequenceStmt.Asc.
func (rs ResultSequence) Asc(column ...string) {
	for i := 0; i < len(column); i++ {
		rs[column[i]] = Asc
	}
}

// Desc implements SequenceStmt.Desc.
func (rs ResultSequence) Desc(column ...string) {
	for i := 0; i < len(column); i++ {
		rs[column[i]] = Desc
	}
}

func generateSelectQuery(b *PqBase, model Model) error {
	var q = "SELECT %s FROM %s"
	f, _ := b.filter.(*SQLFilter)
	slt := "*"
	if len(f.selection) > 0 {
		slt = strings.Join(f.selection, ",")
	}

	if f == nil {
		b.query = fmt.Sprintf("SELECT %s FROM %s;", slt, model.TableName())
		return nil
	}

	q = fmt.Sprintf(q, slt, model.TableName())
	fmt.Println(f.columnValues)

	if len(f.columnValues) > 0 {
		// for where clause here
		q += " WHERE "
	}

	q += b.getSQLWhere()

	if f.limit == 1 {
		q += " LIMIT 1"
	}

	b.query = q + ";"

	return nil
}

func (pb *PqBase) getSQLWhere() string {
	qcount := 1

	and := []string{}
	or := []string{}
	f := pb.filter.(*SQLFilter)
	for _, d := range f.columnValues {
		var w = fmt.Sprintf("%s%s$%d", d.Name, string(d.Condition), qcount)
		if d.LogicalComparison == OR {
			or = append(or, w)
		} else {
			and = append(and, w)
		}

		pb.values = append(pb.values, d.Value)
		qcount++
	}

	whereClause := strings.Join(and, string(AND))
	if len(or) > 0 {
		if whereClause != "" {
			whereClause += fmt.Sprintf(" OR %s", strings.Join(or, string(OR)))
		} else {
			whereClause = strings.Join(or, string(OR))
		}
	}

	return whereClause
}
