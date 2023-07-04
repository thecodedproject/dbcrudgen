package internal

import (
	"errors"
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
				"github.com/thecodedproject/gotest/time",
			)

			modelName := model.Name
			modelStruct, ok := model.Type.(gopkg.TypeStruct)
			if !ok {
				return nil, errors.New("found datamodel which is not of type struct")
			}

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, strcase.ToSnake(model.Name), "db_crud.go"),
				PackageName: dbcrudDir,
				PackageImportPath: dbcrudImport,
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					insertMethod(d, modelName, modelStruct),
					queryByIDMethod(d, modelName, modelStruct),
				},
			})
		}

		return files, nil
	}
}

func insertMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	query := "insert into " + strcase.ToSnake(modelName) + " set "
	queryArgs := make([]string, 0, len(modelStruct.Fields))

	for iF, field := range modelStruct.Fields {

		if field.Name == "ID" {
			continue
		}

		if field.Name == "CreatedAt" || field.Name == "UpdatedAt" {
			queryArgs = append(queryArgs, "time.Now()")
		} else {
			queryArgs = append(queryArgs, "d." + field.Name)
		}

		query += strcase.ToSnake(field.Name) + "=?"

		if iF < len(modelStruct.Fields)-1 {
			query += ", "
		}
	}


	return gopkg.DeclFunc{
		Name: "Insert",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "d",
				Type: gopkg.TypeNamed{
					Name: modelName,
					Import: d.Import.Import,
				},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeInt64{},
			gopkg.TypeError{},
		),
		BodyData: struct{
			DBInsertArgs []string
		}{
			DBInsertArgs: queryArgs,
		},
		BodyTmpl: `
	r, err := db.ExecContext(
		ctx,
		"` + query + `",
{{- range .BodyData.DBInsertArgs}}
		{{.}},
{{- end}}
	)
	if err != nil {
		{{FuncReturnDefaultsWithErr}}
	}

	id, err := r.LastInsertId()
	if err != nil {
		{{FuncReturnDefaultsWithErr}}
	}

	return id, nil
`,
	}
}

func queryByIDMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudDir := strcase.ToSnake(modelName)
	dbcrudImport := path.Join(d.Import.Import, dbcrudDir)

	dbModelType := d.Import.Alias + "." + modelName

	query := "select "
	scanArgs := make([]string, 0, len(modelStruct.Fields))

	for iF, field := range modelStruct.Fields {
		scanArgs = append(scanArgs, "&d." + field.Name)

		_, isBool := field.Type.(gopkg.TypeBool)
		if isBool {
			// The golang sql driver doesn't convert bools nicely
			// Running the select query as:
			//  `select (my_bool = '1') from my_table`
			// is the easiest way I've found to solve the issue
			//
			// See: https://github.com/go-sql-driver/mysql/issues/440
			query += "(" + strcase.ToSnake(field.Name) + " = '1')"
		} else {
			query += strcase.ToSnake(field.Name)
		}

		if iF < len(modelStruct.Fields)-1 {
			query += ", "
		}
	}

	query += " from " + strcase.ToSnake(modelName) + " where id=?"


	return gopkg.DeclFunc{
		Name: "QueryRowByID",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "id",
				Type: gopkg.TypeInt64{},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeNamed{
				Name: dbModelType,
				Import: dbcrudImport,
				ValueType: gopkg.TypeStruct{},
			},
			gopkg.TypeError{},
		),
		BodyData: scanArgs,
		BodyTmpl: `
	var d ` + dbModelType + `

	err := db.QueryRowContext(
		ctx,
		"` + query + `",
		id,
	).Scan(
{{- range .BodyData}}
		{{.}},
{{- end}}
	)
	if err != nil {
		{{FuncReturnDefaultsWithErr}}
	}

	return d, nil
`,
	}
}

func ctxArg() gopkg.DeclVar {
	return gopkg.DeclVar{
		Name: "ctx",
		Type: gopkg.TypeNamed{
			Name: "Context",
			Import: "context",
		},
	}
}

func dbArg() gopkg.DeclVar {
	return gopkg.DeclVar{
		Name: "db",
		Type: gopkg.TypePointer{
			ValueType: gopkg.TypeNamed{
				Name: "DB",
				Import: "database/sql",
			},
		},
	}
}

