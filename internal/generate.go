package internal

import (
	"flag"
	"path"

	"github.com/thecodedproject/gopkg"
	"github.com/thecodedproject/gopkg/tmpl"
)

var (
	outputPath = flag.String("outdir,o", ".", "output directory for generated files")
	// TODO add flags for making methods public
	//publicInsert = flag.Bool("public_insert", "

	useDBContext = flag.Bool("db_context", false, "use DB context in generated methods")
)

type pkgDef struct {
	OutputPath string
	Import gopkg.ImportAndAlias
	DBDataModels []gopkg.DeclType
	PkgTypes []gopkg.DeclType
	UseDBContext bool
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
		PkgTypes: allPkgTypes(currentPkg),
		UseDBContext: *useDBContext,
	}, nil
}

func findDataModels(
	p []gopkg.FileContents,
) ([]gopkg.DeclType, error) {

	dataModelEmbedType := gopkg.TypeNamed{
		Name: "DataModel",
		Import: "github.com/thecodedproject/dbcrudgen/dbcrudgen",
	}

	models := make([]gopkg.DeclType, 0)
	for _, file := range p {
		for _, typeDecl := range file.Types {
			s, isStruct := typeDecl.Type.(gopkg.TypeStruct)
			if isStruct {
				for _, e := range s.Embeds {
					if e == dataModelEmbedType {
						models = append(models, typeDecl)
					}
				}
			}
		}
	}

	return models, nil
}

func allPkgTypes(
	p []gopkg.FileContents,
) []gopkg.DeclType {

	allTypes := make([]gopkg.DeclType, 0)
	for _, f := range p {
		allTypes = append(allTypes, f.Types...)
	}
	return allTypes
}

