package internal

import (
	"flag"
	"path"

	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gopkg/tmpl"

	"fmt"
)

var (
	outputPath = flag.String("outdir,o", ".", "output directory for generated files")
	// TODO add flags for making methods public
	//publicInsert = flag.Bool("public_insert", "
)

type pkgDef struct {
	OutputPath string
	Import gopkg.ImportAndAlias
	DBDataModels []gopkg.DeclType
}

func Generate() error {

	flag.Parse()

	d, err := createPkgDef()
	if err != nil {
		return err
	}

	err = generateSchemaSql(d)
	if err != nil {
		return err
	}

	var files []gopkg.FileContents
	files, err = tmpl.AppendFileContents(
		files,
		fileDBCrud(d),
		fileDBCrudTest(d),
	)
	if err != nil {
		return err
	}

	return gopkg.LintAndGenerate(files)
}

func createPkgDef() (pkgDef, error) {

	importPath, err := gopkg.PackageImportPath(*outputPath)
	if err != nil {
		return pkgDef{}, err
	}

	pkgName := path.Base(importPath)

	currentPkg, err := gopkg.Parse(".")
	if err != nil {
		return pkgDef{}, err
	}

	models, err := findDataModels(currentPkg)
	if err != nil {
		return pkgDef{}, err
	}

	return pkgDef{
		OutputPath: *outputPath,
		Import: gopkg.ImportAndAlias{
			Import: importPath,
			Alias: pkgName,
		},
		DBDataModels: models,
	}, nil
}

func findDataModels(
	p []gopkg.FileContents,
) ([]gopkg.DeclType, error) {

	dataModelEmbedType := gopkg.TypeNamed{
		Name: "DataModel",
		Import: "github.com/thecodedproject/dbcrudgen/dbcrudgen",
	}

	// TODO Not sure what bound to add here so added 100; do better...
	models := make([]gopkg.DeclType, 0, 100)

	for _, file := range p {
		for _, typeDecl := range file.Types {
			s, isStruct := typeDecl.Type.(gopkg.TypeStruct)
			if isStruct {


				for _, e := range s.Embeds {
					fmt.Println(typeDecl.Name, e)
					if e == dataModelEmbedType {
						models = append(models, typeDecl)
					}
				}
			}
		}
	}

	fmt.Println("hello", models)

	return models, nil
}
