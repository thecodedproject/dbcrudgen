package single_type

import (

	"github.com/thecodedproject/dbcrudgen/dbcrudgen"
)

type EntityWithoutTimestamp struct {
	dbcrudgen.DataModel

	ID int64

	AStr string
	BStr string

	AByteSlice []byte

	AInt64 int64
	BInt32 int32

	AFloat32 float32
	BFloat64 float64
}
