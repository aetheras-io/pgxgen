package main

//$tick is a backtick variable used for escaping.  Can't escape a backtick in a backtick
const templateTable = `package {{.PkgName}}
{{$tick := "` + "`" + `"}}
import (
  "context"
  "fmt"
  "strings"

  "github.com/jackc/pgx/v4"
  errors "golang.org/x/xerrors"
)

type {{.StructName}} struct {
{{range .Columns}}  {{.FieldName}} {{.GoType}} {{$tick}}json:",omitempty"{{$tick}}
{{end}}}

{{template "count_func" .}}
{{template "select_all_func" .}}
{{template "select_by_pk_func" .}}
{{template "insert_func" .}}
{{template "update_func" .}}
{{template "delete_func" .}}
`
