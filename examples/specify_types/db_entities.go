package specify_types

import (
	"github.com/thecodedproject/dbcrudgen/dbcrudgen"
)

type AnEntity struct {
	dbcrudgen.DataModel

	ID int64

	AStr string `dbcrudgen:"char(255)"`
	BStr string `dbcrudgen:"varchar(128)"`
}
