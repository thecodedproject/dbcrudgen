package internal

import (
	"path"
	"path/filepath"

	"github.com/iancoleman/strcase"
	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gopkg/tmpl"
)

func fileDBCrud(d pkgDef) func() ([]gopkg.FileContents, error) {

	return func() ([]gopkg.FileContents, error) {

		files := make([]gopkg.FileContents, 0, len(d.DBDataModels))
		for _, model := range d.DBDataModels {

			dbcrudDir := strcase.ToSnake(model.Name)
			dbcrudImport := path.Join(d.Import.Import, dbcrudDir)

			imports := tmpl.UnnamedImports(
			)

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, strcase.ToSnake(model.Name), "db_crud.go"),
				PackageName: dbcrudDir,
				PackageImportPath: dbcrudImport,
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					{
						Name: "Insert",
						Args: []gopkg.DeclVar{
							{
								Name: "ctx",
								Type: gopkg.TypeNamed{
									Name: "Context",
									Import: "context",
								},
							},
							{
								Name: "d",
								Type: gopkg.TypeNamed{
									Name: model.Name,
									Import: d.Import.Import,
								},
							},
						},
						ReturnArgs: tmpl.UnnamedReturnArgs(
							gopkg.TypeInt64{},
							gopkg.TypeError{},
						),
					},
				},
			})
		}

		return files, nil
	}
}

