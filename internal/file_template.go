package internal

import (
	gopkg "github.com/thecodedproject/gopkg"
	tmpl "github.com/thecodedproject/gopkg/tmpl"
	filepath "path/filepath"
)

func fileTemplate(d pkgDef) func() ([]gopkg.FileContents, error) {

	return func() ([]gopkg.FileContents, error) {

		// REMOVE THIS RETURN TO ACTUALLY GENERATE FILES
		return nil, nil

		imports := tmpl.UnnamedImports(
			// ... Add imports ...
		)

		return []gopkg.FileContents{
			{
				Filepath: filepath.Join(d.OutputPath, "some", "path.go"),
				PackageName: d.Import.Alias,
				PackageImportPath: d.Import.Import,
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					{
						Name: "HelloWorld",
					},
				},
			},
		}, nil
	}
}

