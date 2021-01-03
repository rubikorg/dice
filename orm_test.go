package dice

// func TestSeq(t *testing.T) {
// 	s := Seq()
// 	if reflect.TypeOf(s) != reflect.TypeOf(make(ResultSequence)) {
// 		t.Error("Seq() did not return dice.ResultSequence")
// 	}
// }

// func TestSingle(t *testing.T) {
// 	orm.driver = Postgres
// 	f := Single("id", 1)
// 	if reflect.TypeOf(f) != reflect.TypeOf(&SQLFilter{}) {
// 		t.Error("Single not returning SQLFilter for postgres")
// 	}

// 	sqlf, ok := f.(*SQLFilter)
// 	if !ok {
// 		t.Error("cannot coerce FilterStmt to SQLFilter for postgres")
// 	}

// 	if len(sqlf.columnValues) == 0 {
// 		t.Error("Single() did not add the values to SQLFilter.columnValues")
// 	}
// }
