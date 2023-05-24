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
			dbcrudImport := path.Join(d.Import.Import, dbcrudDir)

			imports := tmpl.UnnamedImports(
				"github.com/thecodedproject/sqltest",
			)
			imports = append(imports,
				//gopkg.ImportAndAlias{
				//	Import: dbcrudImport,
				//	Alias: dbcrudDir,
				//},
				d.Import,
			)

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, dbcrudDir, "db_crud_test.go"),
				PackageName: dbcrudDir + "_test",
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
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {

			_ = sqltest.OpenMysql(t, "schema.sql")

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

