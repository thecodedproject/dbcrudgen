package internal

import (
	"errors"
	"path"
	"path/filepath"
	"strings"

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
				"fmt",
				"github.com/stretchr/testify/require",
				"github.com/thecodedproject/sqltest",
				"github.com/thecodedproject/gotest/assert",
			)
			imports = append(imports,
				gopkg.ImportAndAlias{
					Import: dbcrudImport,
					Alias: dbcrudAlias,
				},
				d.Import,
				gopkg.ImportAndAlias{
					Import: "github.com/thecodedproject/gotest/time",
					Alias: "gotest_time",
				},
			)

			if d.UseDBContext {
				imports = append(imports, tmpl.UnnamedImports(
					"github.com/thecodedproject/dbcrudgen/lib",
				)...)
			}

			helpers, err := testHelperMethods(d, modelName, modelStruct)
			if err != nil {
				return nil, err
			}

			files = append(files, gopkg.FileContents{
				Filepath: filepath.Join(d.OutputPath, dbcrudDir, "db_crud_test.go"),
				PackageName: dbcrudAlias + "_test",
				PackageImportPath: dbcrudImport + "_test",
				Imports: imports,
				Functions: append(
					helpers,
					testfuncInsertAndSelect(d, modelName, modelStruct),
					testfuncSelectByID(d, modelName, modelStruct),
					testfuncUpdate(d, modelName, modelStruct),
					testfuncUpdateByID(d, modelName, modelStruct),
					testfuncDelete(d, modelName, modelStruct),
					testfuncDeleteByID(d, modelName, modelStruct),
				),
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
	dbModelType := d.Import.Alias + "." + modelName

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestInsertAndSelect",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		Query map[string]any
		Expected []` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "selects nothing when nothing inserted",
		},
		{
			Name: "insert one and select",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(11),
			},
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(11, 1, now),
			},
		},
		{
			Name: "insert many and select",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(11),
				populateDataModelFromNonce(21),
				populateDataModelFromNonce(31),
				populateDataModelFromNonce(41),
			},
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(11, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(21, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(31, 3, now),
				populateDataModelFromNonceWithIDAndTimestamp(41, 4, now),
			},
		},
		{
			Name: "insert many and select with query",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(22),
				populateDataModelFromNonce(45),
				populateDataModelFromNonce(45),
				populateDataModelFromNonce(1),
				populateDataModelFromNonce(45),
			},
			Query: queryFromNonce(45),
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(45, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(45, 3, now),
				populateDataModelFromNonceWithIDAndTimestamp(45, 5, now),
			},
		},
		{
			Name: "select query field which is not in data model returns error",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(1),
			},
			Query: map[string]any{
				"some_field_not_in_` + modelName + `": 1,
			},
			ExpectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			actual, err := ` + dbcrudAlias + `.Select(` + ctxAndDbArgs + `, test.Query)
			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, len(test.Expected), len(actual))

			for i := range actual {
				assert.LogicallyEqual(t, test.Expected[i], actual[i], fmt.Sprint(i) + "th element not equal")
			}
		})
	}
`,
	}
}

func testfuncSelectByID(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)
	dbModelType := d.Import.Alias + "." + modelName

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestSelectByID",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		ID int64
		Expected ` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "when ID not found returns error",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(100),
				populateDataModelFromNonce(200),
			},
			ID: 12345,
			ExpectErr: true,
		},
		{
			Name: "when ID is found returns row",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(100),
				populateDataModelFromNonce(200),
				populateDataModelFromNonce(300),
			},
			ID: 2,
			Expected: populateDataModelFromNonceWithIDAndTimestamp(200, 2, now),
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			actual, err := ` + dbcrudAlias + `.SelectByID(` + ctxAndDbArgs + `, test.ID)
			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

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

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestUpdate",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		Updates map[string]any
		Query map[string]any
		ExpectedNumUpdates int64
		Expected []` + dbModelType + `
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
			Updates: queryFromNonce(1),
			Query: map[string]any{
				"field_not_in_` + modelName + `": "update",
			},
			ExpectErr: true,
		},
		{
			Name: "update all records",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(123),
				populateDataModelFromNonce(124),
				populateDataModelFromNonce(125),
				populateDataModelFromNonce(126),
			},
			Updates: queryFromNonce(111),
			ExpectedNumUpdates: 4,
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(111, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(111, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(111, 3, now),
				populateDataModelFromNonceWithIDAndTimestamp(111, 4, now),
			},
		},
		{
			Name: "update records with query",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(123),
				populateDataModelFromNonce(125),
				populateDataModelFromNonce(124),
				populateDataModelFromNonce(125),
				populateDataModelFromNonce(126),
				populateDataModelFromNonce(125),
			},
			Updates: queryFromNonce(999),
			Query: queryFromNonce(125),
			ExpectedNumUpdates: 3,
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(123, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(999, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(124, 3, now),
				populateDataModelFromNonceWithIDAndTimestamp(999, 4, now),
				populateDataModelFromNonceWithIDAndTimestamp(126, 5, now),
				populateDataModelFromNonceWithIDAndTimestamp(999, 6, now),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			numUpdates, err := ` + dbcrudAlias + `.Update(` + ctxAndDbArgs + `, test.Updates, test.Query)
			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, test.ExpectedNumUpdates, numUpdates)

			actual, err := ` + dbcrudAlias + `.Select(` + ctxAndDbArgs + `, nil)
			require.NoError(t, err)

			require.Equal(t, len(test.Expected), len(actual))

			for i := range actual {
				assert.LogicallyEqual(t, test.Expected[i], actual[i], fmt.Sprint(i) + "th element not equal")
			}
		})
	}
`,
	}
}

func testfuncUpdateByID(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)
	dbModelType := d.Import.Alias + "." + modelName

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestUpdateByID",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		ID int64
		Updates map[string]any
		Expected []` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "no updates does not error - even if ID does not exist",
		},
		{
			Name: "when there are updates and ID not found throws error",
			ID: 1234,
			Updates: queryFromNonce(1),
			ExpectErr: true,
		},
		{
			Name: "when update field not in schema throws error",
			ID: 1,
			Updates: map[string]any{
				"field_not_in_the_` + dbModelType + `_type": "update",
			},
			ExpectErr: true,
		},
		{
			Name: "insert many and update one by id",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(101),
				populateDataModelFromNonce(102),
				populateDataModelFromNonce(103),
				populateDataModelFromNonce(104),
			},
			ID: 3,
			Updates: queryFromNonce(555),
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(101, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(102, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(555, 3, now),
				populateDataModelFromNonceWithIDAndTimestamp(104, 4, now),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			err := ` + dbcrudAlias + `.UpdateByID(` + ctxAndDbArgs + `, test.ID, test.Updates)

			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			actual, err := ` + dbcrudAlias + `.Select(` + ctxAndDbArgs + `, nil)
			require.NoError(t, err)

			require.Equal(t, len(test.Expected), len(actual))

			for i := range actual {
				assert.LogicallyEqual(t, test.Expected[i], actual[i], fmt.Sprint(i) + "th element not equal")
			}
		})
	}
`,
	}
}

func testfuncDelete(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)
	dbModelType := d.Import.Alias + "." + modelName

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestDelete",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		Query map[string]any
		ExpectedNumDeleted int64
		Expected []` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "empty query deletes all records",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(1000),
				populateDataModelFromNonce(1001),
				populateDataModelFromNonce(1002),
				populateDataModelFromNonce(1003),
				populateDataModelFromNonce(1004),
			},
			ExpectedNumDeleted: 5,
		},
		{
			Name: "delete records using query",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(1000),
				populateDataModelFromNonce(1001),
				populateDataModelFromNonce(1002),
				populateDataModelFromNonce(1002),
				populateDataModelFromNonce(1003),
				populateDataModelFromNonce(1004),
				populateDataModelFromNonce(1002),
			},
			Query: queryFromNonce(1002),
			ExpectedNumDeleted: 3,
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(1000, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(1001, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(1003, 5, now),
				populateDataModelFromNonceWithIDAndTimestamp(1004, 6, now),
			},
		},
		{
			Name: "when query contains field not in data model returns error",
			Query: map[string]any{
				"some_field_not_in_` + modelName + `": 1,
			},
			ExpectErr: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			numDeleted, err := ` + dbcrudAlias + `.Delete(` + ctxAndDbArgs + `, test.Query)
			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, test.ExpectedNumDeleted, numDeleted)

			actual, err := ` + dbcrudAlias + `.Select(` + ctxAndDbArgs + `, nil)
			require.NoError(t, err)

			require.Equal(t, len(test.Expected), len(actual))

			for i := range actual {
				assert.LogicallyEqual(t, test.Expected[i], actual[i], fmt.Sprint(i) + "th element not equal")
			}
		})
	}
`,
	}
}

func testfuncDeleteByID(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) gopkg.DeclFunc {

	dbcrudAlias := strcase.ToSnake(modelName)
	dbModelType := d.Import.Alias + "." + modelName

	ctxAndDbArgs := `ctx, db`
	if d.UseDBContext {
		ctxAndDbArgs = `ctx`
	}

	return gopkg.DeclFunc{
		Name: "TestDeleteByID",
		Args: []gopkg.DeclVar{
			testingArg(),
		},
		BodyData: d,
		BodyTmpl: `
	now := gotest_time.SetTimeNowForTesting(t)

	testCases := []struct{
		Name string
		ToInsert []` + dbModelType + `
		ID int64
		Expected []` + dbModelType + `
		ExpectErr bool
	}{
		{
			Name: "when ID not found returns error",
			ExpectErr: true,
		},
		{
			Name: "insert many and delete by ID",
			ToInsert: []` + dbModelType + `{
				populateDataModelFromNonce(101),
				populateDataModelFromNonce(102),
				populateDataModelFromNonce(103),
				populateDataModelFromNonce(104),
			},
			ID: 3,
			Expected: []` + dbModelType + `{
				populateDataModelFromNonceWithIDAndTimestamp(101, 1, now),
				populateDataModelFromNonceWithIDAndTimestamp(102, 2, now),
				populateDataModelFromNonceWithIDAndTimestamp(104, 4, now),
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			db := sqltest.OpenMysql(t, "schema.sql")
{{- if .BodyData.UseDBContext}}
			ctx := lib.ContextWithDB(context.Background(), db)
{{- else}}
			ctx := context.Background()
{{- end}}

			for _, d := range test.ToInsert {
				_, err := ` + dbcrudAlias + `.Insert(` + ctxAndDbArgs + `, d)
				require.NoError(t, err)
			}

			err := ` + dbcrudAlias + `.DeleteByID(` + ctxAndDbArgs + `, test.ID)

			if test.ExpectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			actual, err := ` + dbcrudAlias + `.Select(` + ctxAndDbArgs + `, nil)
			require.NoError(t, err)

			require.Equal(t, len(test.Expected), len(actual))

			for i := range actual {
				assert.LogicallyEqual(t, test.Expected[i], actual[i], fmt.Sprint(i) + "th element not equal")
			}
		})
	}
`,
	}
}

func testHelperMethods(
	d pkgDef,
	modelName string,
	modelStruct gopkg.TypeStruct,
) ([]gopkg.DeclFunc, error) {

	dbModelType := d.Import.Alias + "." + modelName

	fieldVaules := make(map[string]string)
	specialTimeFields := make([]string, 0, len(modelStruct.Fields))
	for _, f := range modelStruct.Fields {
		specialFields := map[string]bool{
			"ID": true,
			"InsertedAt": true,
			"UpdatedAt": true,
		}

		if specialFields[f.Name] {
			if f.Name != "ID" {
				specialTimeFields = append(specialTimeFields, f.Name)
			}
			continue
		}

		val, err := randomDataForFieldType(f.Type, d.PkgTypes)
		if err != nil {
			return nil, err
		}

		fieldVaules[f.Name] = val
	}

	return []gopkg.DeclFunc{
		{
			Name: "populateDataModelFromNonce",
			Args: []gopkg.DeclVar{
				{
					Name: "nonce",
					Type: gopkg.TypeInt64{},
				},
			},
			ReturnArgs: tmpl.UnnamedReturnArgs(
				gopkg.TypeNamed{
					Name: modelName,
					Import: d.Import.Import,
				},
			),
			BodyData: fieldVaules,
			BodyTmpl: `
	return ` + dbModelType + `{
{{- range $field, $val := .BodyData}}
		{{$field}}: {{$val}},
{{- end}}
	}
`,
		},
		{
			Name: "populateDataModelFromNonceWithIDAndTimestamp",
			Args: []gopkg.DeclVar{
				{
					Name: "nonce",
					Type: gopkg.TypeInt64{},
				},
				{
					Name: "id",
					Type: gopkg.TypeInt64{},
				},
				{
					Name: "t",
					Type: gopkg.TypeNamed{
						Name: "Time",
						Import: "time",
					},
				},
			},
			ReturnArgs: tmpl.UnnamedReturnArgs(
				gopkg.TypeNamed{
					Name: modelName,
					Import: d.Import.Import,
				},
			),
			BodyData: specialTimeFields,
			BodyTmpl: `
	d := populateDataModelFromNonce(nonce)
	d.ID = id
{{- range .BodyData}}
	d.{{.}} = t.Round(gotest_time.Second)
{{- end}}
	return d
`,
		},
		{
			Name: "queryFromNonce",
			Args: []gopkg.DeclVar{
				{
					Name: "nonce",
					Type: gopkg.TypeInt64{},
				},
			},
			ReturnArgs: tmpl.UnnamedReturnArgs(
				gopkg.TypeMap{
					KeyType: gopkg.TypeString{},
					ValueType: gopkg.TypeAny{},
				},
			),
			BodyData: fieldVaules,
			BodyTmpl: `
	return map[string]any{
{{- range $field, $val := .BodyData}}
		"{{ToSnake $field}}": {{$val}},
{{- end}}
	}
`,
		},
	}, nil
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

func randomDataForFieldType(
	goType gopkg.Type,
	enumTypes []gopkg.DeclType,
) (string, error) {

	typeStr, err := goType.FullType(
		map[string]string{
			"time": "time",
		},
	)
	if err != nil {
		return "", err
	}

	switch typeStr {
	case "[]byte":
		return `[]byte("some_bytes" + fmt.Sprint(nonce))`, nil
	case "bool":
		return `nonce%2==0`, nil
	case "float32":
		return `float32(nonce)`, nil
	case "float64":
		return `float64(nonce)`, nil
	case "int32":
		return `int32(nonce)`, nil
	case "int64":
		return `nonce`, nil
	case "string":
		return `"some_str" + fmt.Sprint(nonce)`, nil
	case "time.Time":
		return `time.Unix(nonce, 0)`, nil
	}

	if t, ok := goType.(gopkg.TypeNamed); ok {
		declT, err := findDeclType(t, enumTypes)
		if err != nil {
			return "", err
		}

		d, err := randomDataForFieldType(declT.Type, enumTypes)
		if err != nil {
			return "", err
		}

		importElems := strings.Split(t.Import, "/")

		enumTypeStr, err := t.FullType(
			map[string]string{
				t.Import: importElems[len(importElems)-1],
			},
		)
		if err != nil {
			return "", err
		}

		return enumTypeStr + "(" + d + ")", nil
	}

	return "", errors.New("cannot generate DB tests for go type " + typeStr)
}
