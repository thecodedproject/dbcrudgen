package internal

import (
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
					{
						Name: "TestInsertSingleAndQuery",
						Args: []gopkg.DeclVar{
							{
								Name: "t",
								Type: gopkg.TypePointer{
									ValueType: gopkg.TypeNamed{
										Name: "T",
										Import: "testing",
									},
								},
							},
						},
						BodyData: struct{
							ModelType string
						}{
							ModelType: d.Import.Alias + "." + model.Name,
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

			actual, err := ` + dbcrudAlias + `.QueryRowByID(ctx, db, id)
			require.NoError(t, err)

			test.Expected.ID = id

			now = now.Round(time.Second)

			test.Expected.CreatedAt = now
			test.Expected.UpdatedAt = now

			assert.LogicallyEqual(t, test.Expected, actual)
		})
	}
`,
					},
				},
			})
		}

		return files, nil
	}
}

