package enum_types

import (
	"github.com/thecodedproject/dbcrudgen/dbcrudgen"
)

const (
	Int32EnumUnknown Int32Enum = 0
	Int32EnumOne Int32Enum = 1
	Int32EnumTwo Int32Enum = 2

	Int64EnumUnknown Int64Enum = 0
	Int64EnumOne Int64Enum = 1
	Int64EnumTwo Int64Enum = 2

	StringEnumUnknown StringEnum = "String_unknown"
	StringEnumOne StringEnum = "String_one"
	StringEnumTwo StringEnum = "String_two"
)

var (
	ByteArrayEnumUnknown ByteArrayEnum = []byte("ByteArray_unknown")
	ByteArrayEnumOne ByteArrayEnum = []byte("ByteArray_one")
	ByteArrayEnumTwo ByteArrayEnum = []byte("ByteArray_two")
)


type ByteArrayEnum []byte
type Int32Enum int32
type Int64Enum int64
type IntEnum int
type StringEnum string


type ByteArrayData struct {
	dbcrudgen.DataModel
	ID int64
	Enum ByteArrayEnum
}

type Int32Data struct {
	dbcrudgen.DataModel
	ID int64
	Enum Int32Enum
}

type Int64Model struct {
	dbcrudgen.DataModel
	ID int64
	Enum Int64Enum
}

type StringModel struct {
	dbcrudgen.DataModel
	ID int64
	Enum StringEnum
}
