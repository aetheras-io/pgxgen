package main

const templateCountFunc = `const count{{.StructName}}SQL = {{getTick}}select count(*) from {{.TableName}}{{getTick}}

func Count{{.StructName}}(ctx context.Context, db Queryer) (int64, error) {
  var n int64
  err := prepareQueryRow(ctx, db, "pgxgenCount{{.StructName}}", count{{.StructName}}SQL).Scan(&n)
  return n, err
}
`

const templateInsertFunc = `func Insert{{.StructName}}(ctx context.Context, db Queryer, row *{{.StructName}}) error {  
	var columns, values []string
	args := pgx.QueryArgs(make([]interface{}, 0, {{len .Columns}}))
	{{range .Columns}} {{$prefix := "row."}}
	if {{ goNullComparison (print $prefix .FieldName) .GoType }} {
		columns = append(columns, "{{.ColumnName}}")
		values = append(values, args.Append(&row.{{.FieldName}}))
	}{{end}}
	
	sql := fmt.Sprintf({{getTick}}insert into users(%s) values(%s) returning id{{getTick}}, strings.Join(columns, ", "), strings.Join(values, ","))  
	psName := preparedName("pgxgenInsert{{.StructName}}", sql)
  
	return prepareQueryRow(ctx, db, psName, sql, args...).Scan({{ range $i, $column := .PrimaryKeyColumns}}{{if $i}}, {{end}}&row.{{$column.FieldName}}{{end}})
}
`

const templateSelectByPKFunc = `const select{{.StructName}}ByPKSQL = {{getTick}}select{{ range $i, $column := .Columns}}{{if $i}},{{end}}
{{$column.ColumnName}}{{end}}
from {{.TableName}}
where {{ range $i, $column := .PrimaryKeyColumns}}{{if $i}} and {{end}}{{$column.ColumnName}}={{pkPlaceholder $i}}{{end}}{{getTick}}

func Select{{.StructName}}ByPK(
	ctx context.Context,
	db Queryer{{range .PrimaryKeyColumns}},
	{{.VarName}} {{.GoType}}{{end}},
	) (*{{.StructName}}, error) {
	var row {{.StructName}}
	err := prepareQueryRow(ctx, db, "pgxgenSelect{{.StructName}}ByPK", select{{.StructName}}ByPKSQL{{range .PrimaryKeyColumns}}, {{.VarName}}{{end}}).Scan(
	{{range .Columns}}&row.{{.FieldName}},
  	{{end}})
	if errors.Is(err, pgx.ErrNoRows) {
	  return nil, ErrNotFound
	} else if err != nil {
	  return nil, err
	}

	return &row, nil
}
`

const templateSelectAllFunc = `const SelectAll{{.StructName}}SQL = {{getTick}}select{{ range $i, $column := .Columns}}{{if $i}},{{end}}
{{$column.ColumnName}}{{end}}
from {{.TableName}}{{getTick}}

func SelectAll{{.StructName}}(ctx context.Context, db Queryer) ([]{{.StructName}}, error) {
var rows []{{.StructName}}
	dbRows, err := prepareQuery(ctx, db, "pgxgenSelectAll{{.StructName}}", SelectAll{{.StructName}}SQL)
	if err != nil {
	  return nil, err
	}

	for dbRows.Next() {
	  var row {{.StructName}}
	  dbRows.Scan(
	{{range .Columns}}&row.{{.FieldName}},
	  {{end}})
	  rows = append(rows, row)
	}

	if dbRows.Err() != nil {
	  return nil, dbRows.Err()
	}

	return rows, nil
	}
`
const templateUpdateFunc = `func Update{{.StructName}}(ctx context.Context, db Queryer{{range .PrimaryKeyColumns}},
	{{.VarName}} {{.GoType}}{{end}},
	row *{{.StructName}},
  ) error {
	sets := make([]string, 0, {{len .Columns}})
	args := pgx.QueryArgs(make([]interface{}, 0, {{len .Columns}}))
	{{range .Columns}} {{$prefix := "row."}}
	if {{ goNullComparison (print $prefix .FieldName) .GoType }} {
	  sets = append(sets, "{{.ColumnName}}"+"="+args.Append(&row.{{.FieldName}}))
	}{{end}}

	if len(sets) == 0 {
	  return nil
	}
  
	sql := fmt.Sprintf({{getTick}}update users set %s where id=%s{{getTick}}, strings.Join(sets, ", "), args.Append(id))  
	psName := preparedName("pgxgenUpdate{{.StructName}}", sql)
  
	commandTag, err := prepareExec(ctx, db, psName, sql, args...)
	if err != nil {
	  return err
	}
	if commandTag.RowsAffected() != 1 {
	  return ErrNotFound
	}
	return nil
}
`

const templateDeleteFunc = `func Delete{{.StructName}}(ctx context.Context, db Queryer{{range .PrimaryKeyColumns}},
	{{.VarName}} {{.GoType}}{{end}},
  ) error {
	args := pgx.QueryArgs(make([]interface{}, 0, {{len .PrimaryKeyColumns}}))
	sql := {{getTick}}delete from {{.TableName}} where {{ range $i, $column := .PrimaryKeyColumns}}{{getTick}} + {{getTick}}{{if $i}} and {{end}}{{$column.ColumnName}}={{getTick}} + args.Append({{$column.VarName}}){{end}}
  
	commandTag, err := prepareExec(ctx, db, "pgxgenDelete{{.StructName}}", sql, args...)
	if err != nil {
	  return err
	}
	if commandTag.RowsAffected() != 1 {
	  return ErrNotFound
	}
	return nil
}
`
