package internal

import (
	"errors"
	"path"
	"path/filepath"

	"github.com/iancoleman/strcase"
	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gopkg/tmpl"
)

func fileDBCrudTest(d pkgDef) func() ([]gopkg.FileContents, error) {

	return func() ([]gopkg.FileContents, error) {

		files := make([]gopkg.FileContents, 0, len(d.DBDataModels))
		for _, model := range d.DBDataModels {

			dbcrudDir := strcase.ToSnake(model.Name)
			dbcrudAlias := dbcrudDir
			dbcrudImport := path.Join(d.Import.Import, dbcrudDir)

			modelName := model.Name
			modelStruct, ok := model.Type.(gopkg.TypeStruct)
			if !ok {
				return nil, errors.New("found datamodel which is not of type struct")
			}

			imports := tmpl.UnnamedImports(
				"context",
				"github.com/stretchr/testify/require",
				"github.com/thecodedproject/sqltest",
				"github.com/thecodedproject/gotest/assert",
				"github.com/thecodedproject/gotest/time",
			)
			imports = append(imports,
				gopkg.ImportAndAlias{
					Import: dbcrudImport,
					Alias: dbcrudAlias,
				},
				d.Import,
			)

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, dbcrudDir, "db_crud_test.go"),
				PackageName: dbcrudAlias + "_test",
				PackageImportPath: dbcrudImport + "_test",
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					testfuncInsertAndSelect(d, modelName, modelStruct),
					testfuncUpdate(d, modelName, modelStruct),
				},
			})
		}

		return files, nil
	}
}

func testfuncInsertAndSelect(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)

	return gopkg.DeclFunc{
		Name: "TestInsertSingleAndSelect",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: struct{
			ModelType string
		}{
			ModelType: d.Import.Alias + "." + modelName,
		},
		BodyTmpl: `
	testCases := []struct{
		Name string
		Data {{.BodyData.ModelType}}
		Expected {{.BodyData.ModelType}}
	}{
		{
			Name: "Insert empty and query",
		},
		{
			Name: "Insert data and query",
			Data: {{.BodyData.ModelType}}{
				SomeString: "abcd",
				SomeInt: 1234,
				SomeBool: true,
			},
			Expected: {{.BodyData.ModelType}}{
				SomeString: "abcd",
				SomeInt: 1234,
				SomeBool: true,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			now := time.SetTimeNowForTesting(t)

			ctx := context.Background()
			db := sqltest.OpenMysql(t, "schema.sql")

			id, err := ` + dbcrudAlias + `.Insert(ctx, db, test.Data)
			require.NoError(t, err)

			actual, err := ` + dbcrudAlias + `.SelectByID(ctx, db, id)
			require.NoError(t, err)

			test.Expected.ID = id

			now = now.Round(time.Second)

			test.Expected.CreatedAt = now
			test.Expected.UpdatedAt = now

			assert.LogicallyEqual(t, test.Expected, actual)
		})
	}
`,
	}
}

func testfuncUpdate(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)
	dbModelType := d.Import.Alias + "." + modelName

	var fieldToUpdate string
	for _, f := range modelStruct.Fields {
		specialFields := map[string]bool{
			"ID": true,
			"CreatedAt": true,
			"UpdatedAt": true,
		}

		if !specialFields[f.Name] {
			fieldToUpdate = strcase.ToSnake(f.Name)
			break
		}
	}

	return gopkg.DeclFunc{
		Name: "TestUpdate",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		//BodyData: scanArgs,
		BodyTmpl: `

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		Updates map[string]any
		Query map[string]any
		ExpectedNumUpdates int64
		ExpectedData []` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "empty params does nothing",
		},
		{
			Name: "update unknown field throws error",
			Updates: map[string]any{
				"field_not_in_the_` + dbModelType + `_type": "update",
			},
			ExpectErr: true,
		},
		{
			Name: "query unknown field throws error",
			Updates: map[string]any{
				"` + fieldToUpdate + `": "1",
			},
			Query: map[string]any{
				"field_not_in_the_` + dbModelType + `_type": "update",
			},
			ExpectErr: true,
		},
		{
			Name: "update all records",
			ToInsert: make([]` + dbModelType + `, 2),
			Updates: map[string]any{
				"` + fieldToUpdate + `": "1",
			},
			ExpectedNumUpdates: 2,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			ctx := context.Background()
			db := sqltest.OpenMysql(t, "schema.sql")

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(
					ctx,
					db,
					d,
				)
				require.NoError(t, err)
			}

			numUpdates, err := ` + dbcrudAlias + `.Update(
				ctx,
				db,
				test.Updates,
				test.Query,
			)

			if test.ExpectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.ExpectedNumUpdates, numUpdates)
		})
	}
`,
	}
}

func testingArg() gopkg.DeclVar {
	return gopkg.DeclVar{
		Name: "t",
		Type: gopkg.TypePointer{
			ValueType: gopkg.TypeNamed{
				Name: "T",
				Import: "testing",
			},
		},
	}
}
