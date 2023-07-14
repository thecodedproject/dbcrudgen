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
				"errors",
				"fmt",
			)

			modelName := model.Name
			modelStruct, ok := model.Type.(gopkg.TypeStruct)
			if !ok {
				return nil, errors.New("found datamodel which is not of type struct")
			}

			for _, field := range modelStruct.Fields {
				if field.Name == "InsertedAt" || field.Name == "UpdatedAt" {
					imports = append(
						imports,
						gopkg.ImportAndAlias{
							Import: "github.com/thecodedproject/gotest/time",
						},
					)
					break
				}
			}

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, strcase.ToSnake(model.Name), "db_crud.go"),
				PackageName: dbcrudDir,
				PackageImportPath: dbcrudImport,
				Imports: imports,
				Functions: []gopkg.DeclFunc{
					insertMethod(d, modelName, modelStruct),
					selectByIDMethod(d, modelName, modelStruct),
					selectMethod(d, modelName, modelStruct),
					updateMethod(d, modelName, modelStruct),
					updateByIDMethod(d, modelName, modelStruct),
					deleteMethod(d, modelName, modelStruct),
					deleteByIDMethod(d, modelName, modelStruct),
					modelContainsFieldMethod(d, modelName, modelStruct),
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

		if field.Name == "InsertedAt" || field.Name == "UpdatedAt" {
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

func selectByIDMethod(
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
		Name: "SelectByID",
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
	r, err := Select(
		ctx,
		db,
		map[string]any{
			"id": id,
		},
	)
	if err != nil {
		return ` + dbModelType + `{}, nil
	}

	if len(r) == 0 {
		return ` + dbModelType + `{}, errors.New("SelectByID: id not found - " + fmt.Sprint(id))
	}

	if len(r) > 1 {
		return ` + dbModelType + `{}, errors.New("found more than one entry with id")
	}

	return r[0], nil
`,
	}
}

func selectMethod(
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

	query += " from " + strcase.ToSnake(modelName)


	return gopkg.DeclFunc{
		Name: "Select",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "queryParams",
				Type: gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeArray{
				ValueType: gopkg.TypeNamed{
					Name: dbModelType,
					Import: dbcrudImport,
					ValueType: gopkg.TypeStruct{},
				},
			},
			gopkg.TypeError{},
		),
		BodyData: scanArgs,
		BodyTmpl: `

	q := "` + query + `"

	if len(queryParams) > 0 {
		q += " where "
	}

	queryVals := make([]any, 0, len(queryParams))
	i := 0
	for k, v := range queryParams {
		if !modelContainsField(k) {
			return nil, errors.New("Select: no such field to query - " + k)
		}

		q += k + "=?"
		i++
		if i < len(queryParams) {
			q += " and "
		}
		queryVals = append(queryVals, v)
	}

	r, err := db.QueryContext(
		ctx,
		q,
		queryVals...,
	)
	if err != nil {
		return nil, nil
	}

	// TODO: make this a configurable param
	maxResponses := 1000
	res := make([]` + dbModelType + `, 0, maxResponses)
	for r.Next() {

		if len(res) >= maxResponses {
			return nil, errors.New("select query exceeded max responses")
		}

		var d ` + dbModelType + `
		r.Scan(
{{- range .BodyData}}
			{{.}},
{{- end}}
		)
		if err != nil {
			return nil, nil
		}

		res = append(res, d)
	}

	return res, nil
`,
	}
}

func updateMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbTable := strcase.ToSnake(modelName)

	return gopkg.DeclFunc{
		Name: "Update",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "updates",
				Type: gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			},
			{
				Name: "queryParams",
				Type: gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeInt64{},
			gopkg.TypeError{},
		),
		BodyTmpl: `
	if len(updates) == 0 {
		return 0, nil
	}

	query := "update ` + dbTable + ` set "
	queryArgs := make([]any, 0, len(updates) + len(queryParams))
	i := 0
	for k, v := range updates {
		if !modelContainsField(k) {
			return 0, errors.New("Update: no such field to update - " + k)
		}

		query += k + "=?"
		i++
		if i < len(updates) {
			query += ", "
		}

		queryArgs = append(queryArgs, v)
	}

	if len(queryParams) > 0 {
		query += " where "
	}
	i = 0
	for k, v := range queryParams {
		if !modelContainsField(k) {
			return 0, errors.New("Update: no such field to query - " + k)
		}

		query += k + "=?"
		i++
		if i < len(queryParams) {
			query += " and "
		}

		queryArgs = append(queryArgs, v)
	}

	r, err := db.ExecContext(
		ctx,
		query,
		queryArgs...,
	)
	if err != nil {
		return 0, err
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
`,
	}
}

func updateByIDMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	return gopkg.DeclFunc{
		Name: "UpdateByID",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "id",
				Type: gopkg.TypeInt64{},
			},
			{
				Name: "updates",
				Type: gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeError{},
		),
		BodyTmpl: `
	if len(updates) == 0 {
		return nil
	}

	n, err := Update(
		ctx,
		db,
		updates,
		map[string]any{
			"id": id,
		},
	)
	if err != nil {
		return err
	}

	if n == 0 {
		return errors.New("UpdateByID: no such ID")
	}

	return nil
`,
	}
}

func deleteMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbTable := strcase.ToSnake(modelName)

	return gopkg.DeclFunc{
		Name: "Delete",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "queryParams",
				Type: gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeInt64{},
			gopkg.TypeError{},
		),
		BodyTmpl: `
	query := "delete from ` + dbTable + `"

	if len(queryParams) > 0 {
		query += " where "
	}
	i := 0
	queryArgs := make([]any, 0, len(queryParams))
	for k, v := range queryParams {
		if !modelContainsField(k) {
			return 0, errors.New("Delete: no such field to query - " + k)
		}

		query += k + "=?"
		i++
		if i < len(queryParams) {
			query += " and "
		}

		queryArgs = append(queryArgs, v)
	}

	r, err := db.ExecContext(
		ctx,
		query,
		queryArgs...,
	)
	if err != nil {
		return 0, err
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, err
	}

	return count, nil
`,
	}
}

func deleteByIDMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	return gopkg.DeclFunc{
		Name: "DeleteByID",
		Args: []gopkg.DeclVar{
			ctxArg(),
			dbArg(),
			{
				Name: "id",
				Type: gopkg.TypeInt64{},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeError{},
		),
		BodyTmpl: `
	n, err := Delete(
		ctx,
		db,
		map[string]any{
			"id": id,
		},
	)
	if err != nil {
		return err
	}

	if n == 0 {
		return errors.New("DeleteByID: no such ID")
	}

	return nil
`,
	}
}

func modelContainsFieldMethod(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	modelFields := make([]string, 0, len(modelStruct.Fields))
	for _, f := range modelStruct.Fields {
		modelFields = append(
			modelFields,
			strcase.ToSnake(f.Name),
		)
	}


	return gopkg.DeclFunc{
		Name: "modelContainsField",
		Args: []gopkg.DeclVar{
			{
				Name: "field",
				Type: gopkg.TypeString{},
			},
		},
		ReturnArgs: tmpl.UnnamedReturnArgs(
			gopkg.TypeBool{},
		),
		BodyData: modelFields,
		BodyTmpl: `
	modelFields := map[string]bool{
{{- range .BodyData}}
		"{{.}}": true,
{{- end}}
	}

	return modelFields[field]
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

