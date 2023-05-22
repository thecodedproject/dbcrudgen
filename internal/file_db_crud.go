package internal

import (
	"path/filepath"

	"github.com/iancoleman/strcase"
	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gopkg/tmpl"
)

func fileDBCrud(d pkgDef) func() ([]gopkg.FileContents, error) {

	return func() ([]gopkg.FileContents, error) {

		files := make([]gopkg.FileContents, 0, len(d.DBDataModels))
		for _, model := range d.DBDataModels {

			imports := tmpl.UnnamedImports(
			)

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, strcase.ToSnake(model.Name), "db_crud.go"),
				PackageName: d.Import.Alias,
				PackageImportPath: d.Import.Import,
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					{
						Name: "HelloWorld",
					},
				},
			})
		}

		return files, nil
	}
}

