package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type boxType struct {
	Name       string
	ValueType  string
	FormatCode string
}

type intBoxType struct {
	Name    string
	BitSize int
}

type initData struct {
	PkgName string
	Version string
}

var pgToBoxTypeMap = map[string]string{
	"bigint":                   "pgtype.Int8",
	"integer":                  "pgtype.Int4",
	"smallint":                 "pgtype.Int2",
	"character varying":        "pgtype.Varchar",
	"text":                     "pgtype.Text",
	"date":                     "pgtype.Date",
	"timestamp with time zone": "pgtype.Timestamptz",
	"inet":                     "pgtype.Inet",
	"cidr":                     "pgtype.Cidr",
	"bytea":                    "pgtype.Bytea",
}

var pgToGoTypeMap = map[string]string{
	"bigint":                   "int64",
	"integer":                  "int32",
	"smallint":                 "int16",
	"character varying":        "string",
	"text":                     "string",
	"date":                     "time.Time",
	"timestamp with time zone": "time.Time",
	"bytea":                    "[]byte",
}

var goTypeToNullComparisonFMap = map[string]string{
	"int64":     "%s < 0",
	"int32":     "%s < 0",
	"int16":     "%s < 0",
	"string":    `%s == ""`,
	"time.Time": "%s.IsZero()",
	"[]byte":    "len(%s) == 0",
}

var acronyms = map[string]bool{
	"id":  true,
	"ip":  true,
	"url": true,
}

type Config struct {
	Package string
	Tables  []Table
}

type Column struct {
	ColumnName      string
	DataType        string
	OrdinalPosition int32

	FieldName string
	GoBoxType string

	VarName string
	GoType  string
}

type ColumnConfig struct {
	ColumnName string `toml:"column_name"`
	FieldName  string `toml:"field_name"`
}

type Table struct {
	TableName             string         `toml:"table_name"`
	StructName            string         `toml:"struct_name"`
	PrimaryKeyColumnNames []string       `toml:"primary_key"`
	ColumnConfigs         []ColumnConfig `toml:"columns"`
	Columns               []Column
	PrimaryKeyColumns     []*Column
}

func generateCmd(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "generate does not take any arguments")
		os.Exit(1)
	}

	var c Config
	_, err := toml.DecodeFile("config.toml", &c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pgurl := viper.GetString(FlagPGURL)
	conn, err := pgx.Connect(context.Background(), pgurl)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = inspectDatabase(conn, c.Tables)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	supportData := initData{
		PkgName: c.Package,
		Version: Version,
	}

	supportFiles := []struct {
		path string
		tmpl *template.Template
	}{
		{c.Package + ".go", templates.Lookup("db")},
	}
	for _, f := range supportFiles {
		err := writeTemplateFile(f.path, f.tmpl, supportData)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	for _, t := range c.Tables {
		file, err := os.Create(c.Package + "_" + goCaseToFileCase(t.StructName) + ".go")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		err = writeTableCrud(file, templates, c.Package, t)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		file.Close()
	}
}

type Queryer interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
}

func writeTableCrud(w io.Writer, templates *template.Template, pkgName string, table Table) error {
	return templates.ExecuteTemplate(w, "table", struct {
		PkgName           string
		TableName         string
		StructName        string
		Columns           []Column
		PrimaryKeyColumns []*Column
	}{
		PkgName:           pkgName,
		TableName:         table.TableName,
		StructName:        table.StructName,
		Columns:           table.Columns,
		PrimaryKeyColumns: table.PrimaryKeyColumns,
	})
}

func inspectDatabase(db Queryer, tables []Table) error {
	for i := range tables {
		rows, err := db.Query(context.Background(), `select column_name, data_type, ordinal_position from information_schema.columns where table_name=$1`, tables[i].TableName)
		if err != nil {
			return err
		}

		var columns []Column
		for rows.Next() {
			var c Column
			rows.Scan(&c.ColumnName, &c.DataType, &c.OrdinalPosition)
			c.FieldName = pgCaseToGoPublicCase(c.ColumnName)
			c.GoBoxType = pgTypeToGoBoxType(c.DataType)
			c.VarName = pgCaseToGoPrivateCase(c.ColumnName)
			c.GoType = pgTypeToGoType(c.DataType)

			columns = append(columns, c)
		}

		if rows.Err() != nil {
			return err
		}

		tables[i].Columns = columns

		if len(tables[i].PrimaryKeyColumnNames) == 0 {
			tables[i].PrimaryKeyColumnNames = []string{"id"}
		}

		for _, columnName := range tables[i].PrimaryKeyColumnNames {
			var found bool
			for j := range tables[i].Columns {
				if tables[i].Columns[j].ColumnName == columnName {
					tables[i].PrimaryKeyColumns = append(tables[i].PrimaryKeyColumns, &tables[i].Columns[j])
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("table %s primary_key column %s not found", tables[i].TableName, columnName)
			}
		}

		for _, cc := range tables[i].ColumnConfigs {
			var found bool
			for j := range tables[i].Columns {
				if tables[i].Columns[j].ColumnName == cc.ColumnName {
					if cc.FieldName != "" {
						tables[i].Columns[j].FieldName = cc.FieldName
					}
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("table %s column %s not found", tables[i].TableName, cc.ColumnName)
			}
		}
	}

	return nil
}

func pgCaseToGoPublicCase(pg string) string {
	parts := strings.Split(pg, "_")
	buf := &bytes.Buffer{}
	for _, s := range parts {
		if acronyms[s] {
			s = strings.ToUpper(s)
		} else {
			s = strings.Title(s)
		}
		buf.WriteString(s)
	}

	return buf.String()
}

func pgCaseToGoPrivateCase(pg string) string {
	parts := strings.Split(pg, "_")
	buf := &bytes.Buffer{}
	for i, s := range parts {
		if i == 0 {
			s = strings.ToLower(s)
		} else if acronyms[s] {
			s = strings.ToUpper(s)
		} else {
			s = strings.Title(s)
		}
		buf.WriteString(s)
	}

	return buf.String()
}

func goCaseToFileCase(g string) string {
	buf := &bytes.Buffer{}

	for i, r := range g {
		if unicode.IsUpper(r) && i != 0 {
			buf.WriteRune('_')
		}
		buf.WriteRune(unicode.ToLower(r))
	}

	return buf.String()
}

func pgTypeToGoBoxType(pg string) string {
	if t, ok := pgToBoxTypeMap[pg]; ok {
		return t
	} else {
		return "pgtype.Varchar"
	}
}

func pgTypeToGoType(pg string) string {
	if t, ok := pgToGoTypeMap[pg]; ok {
		return t
	} else {
		return "string"
	}
}

func goTypeToNullComparison(fieldName string, goType string) string {
	if format, ok := goTypeToNullComparisonFMap[goType]; ok {
		return fmt.Sprintf(format, fieldName)
	} else {
		panic("cannot find go type null comparison formatter for :" + goType)
	}
}
