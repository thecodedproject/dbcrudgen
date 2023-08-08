package internal

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"

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
			sqlField, err := makeSqlField(f, d.PkgTypes)
			if err != nil {
				return errors.Wrap(
					err,
					fmt.Sprintf("error making sql field for '%s.%s'", m.Name, f.Name),
				)
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
	enumTypes []gopkg.DeclType,
) (gosql.Field, error) {

	var sqlType gosql.Type
	if t := goField.StructTag.Get("dbcrudgen"); t != "" {
		var err error
		sqlType, err = gosql.ParseType(t)
		if err != nil {
			return gosql.Field{}, err
		}
	} else {
		var err error
		sqlType, err = sqlTypeFromGoType(goField.Type, enumTypes)
		if err != nil {
			return gosql.Field{}, err
		}
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
	enumTypes []gopkg.DeclType,
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
	case "[]byte":
		return gosql.TypeVarChar{N: 255}, nil
	case "bool":
		return gosql.TypeBit{}, nil
	case "float32":
		return gosql.TypeFloat{}, nil
	case "float64":
		return gosql.TypeDouble{}, nil
	case "int32":
		return gosql.TypeInt{}, nil
	case "int64":
		return gosql.TypeBigInt{}, nil
	case "string":
		return gosql.TypeVarChar{N: 255}, nil
	case "time.Time":
		return gosql.TypeDateTime{}, nil
	}

	if t, ok := goType.(gopkg.TypeNamed); ok {
		declT, err := findDeclType(t, enumTypes)
		if err != nil {
			return nil, err
		}

		return sqlTypeFromGoType(declT.Type, enumTypes)
	}

	return nil, errors.New("no conversion from go type `" + typeStr + "` to sql type")
}

func findDeclType(
	t gopkg.TypeNamed,
	declTypes []gopkg.DeclType,
) (gopkg.DeclType, error) {

	for _, d := range declTypes {
		if d.Name == t.Name && d.Import == t.Import {
			return d, nil
		}
	}

	return gopkg.DeclType{}, errors.New(
		fmt.Sprintf(
			"cannot find declaration for type '%s'",
			t.Name,
		),
	)
}
