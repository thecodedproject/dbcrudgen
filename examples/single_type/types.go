package single_type

import (
	"time"

	"github.com/thecodedproject/dbcrudgen/dbcrudgen"
)

type MyDataModel struct {
	dbcrudgen.DataModel

	ID int64
	InsertedAt time.Time
	UpdatedAt time.Time
	SomeString string
	SomeInt int64
	SomeBool bool
}
