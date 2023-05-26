package internal

import (
	"errors"
	"path/filepath"

	"github.com/iancoleman/strcase"
	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gosql"
)

func generateSchemaSql(
	d pkgDef,
) error {

	for _, m := range d.DBDataModels {

		tableSchema := gosql.CreateTable{
			Name: strcase.ToSnake(m.Name),
		}

		mStruct, ok := m.Type.(gopkg.TypeStruct)
		if !ok {
			return errors.New("found datamodel which is not of type struct")
		}

		for _, f := range mStruct.Fields {
			sqlField, err := makeSqlField(f)
			if err != nil {
				return err
			}

			tableSchema.Fields = append(tableSchema.Fields, sqlField)
		}

		err := gosql.GenerateSchema(
			filepath.Join(d.OutputPath, strcase.ToSnake(m.Name), "schema.sql"),
			[]gosql.Statement{
				tableSchema,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func makeSqlField(
	goField gopkg.DeclVar,
) (gosql.Field, error) {

	sqlType, err := sqlTypeFromGoType(goField.Type)
	if err != nil {
		return gosql.Field{}, err
	}

	fieldName := strcase.ToSnake(goField.Name)
	var primaryKey bool
	if fieldName == "id" {
		primaryKey = true
	}

	return gosql.Field{
		Name: fieldName,
		Type: sqlType,
		PrimaryKey: primaryKey,
		AutoIncrement: primaryKey,
	}, nil
}

func sqlTypeFromGoType(
	goType gopkg.Type,
) (gosql.Type, error) {

	typeStr, err := goType.FullType(
		map[string]string{
			"time": "time",
		},
	)
	if err != nil {
		return nil, err
	}

	switch typeStr {
	case "bool":
		return gosql.TypeBit{}, nil
	case "int64":
		return gosql.TypeBigInt{}, nil
	case "string":
		return gosql.TypeVarChar{N: 255}, nil
	case "time.Time":
		return gosql.TypeDateTime{}, nil
	default:
		return nil, errors.New("no conversion from go type `" + typeStr + "` to sql type")
	}
}
