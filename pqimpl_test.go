package dice

import (
	"testing"
)

func TestSQLFilterField(t *testing.T) {
	f := &SQLFilter{}
	cdh := f.Field("test")
	if cdh.column != "test" {
		t.Error("column data header storing wrong column name: " + cdh.column)
	}
}

func TestSQLFilterOr(t *testing.T) {
	f := &SQLFilter{}
	cdh := f.Or("test")
	if cdh.column != "test" {
		t.Error("column data header storing wrong column name: " + cdh.column)
	}

	if cdh.logicalComparison != OR {
		t.Error("column data header has different logical operator for OR query: " +
			cdh.logicalComparison)
	}
}

func TestColumnDataHolder(t *testing.T) {
	f := &SQLFilter{}
	cdh := f.Field("test")
	if cdh.column != "test" {
		t.Error("column data header storing wrong column name: " + cdh.column)
	}

	cdh.Must(Gt, 1)
	if cdh.logicalComparison != AND {
		t.Error("column data header Field() should always be AND: " + cdh.logicalComparison)
	}

	if len(f.columnValues) == 0 {
		t.Error("columnDataHolder.Must() did not add it to SQLFilter.columnValues")
	}

	if len(f.columnValues) > 0 {
		cv := f.columnValues[0]
		val := cv.Value.(int)
		if val != 1 {
			t.Errorf("columnDataHolder.Must() did not store correct value, value must be 1: %v",
				cv.Value)
		}

		if cv.Condition != Gt {
			t.Errorf("columnDataHolder.Must() did not store correct condition: %v", cv.Condition)
		}
	}
}
