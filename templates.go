package main

import (
	"fmt"
	"text/template"
)

func pkPlaceholder(n int) string {
	return fmt.Sprintf("$%d", n+1)
}

func loadTemplates() *template.Template {
	funcMap := template.FuncMap{
		"pkPlaceholder":    pkPlaceholder,
		"goNullComparison": goTypeToNullComparison,
		"goComparison":     goTypeToComparison,
		"getTick":          func() string { return "`" },
	}
	templates := template.New("base").Funcs(funcMap)

	_, err := templates.New(`config`).Parse(templateConfig)
	if err != nil {
		fmt.Println(templateConfig)
		panic("Unable to parse template")
	}

	_, err = templates.New(`count_func`).Parse(templateCountFunc)
	if err != nil {
		fmt.Println(templateCountFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`insert_func`).Parse(templateInsertFunc)
	if err != nil {
		fmt.Println(templateCountFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`select_by_pk_func`).Parse(templateSelectByPKFunc)
	if err != nil {
		fmt.Println(templateSelectByPKFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`update_func`).Parse(templateUpdateFunc)
	if err != nil {
		fmt.Println(templateUpdateFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`delete_func`).Parse(templateDeleteFunc)
	if err != nil {
		fmt.Println(templateDeleteFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`select_all_func`).Parse(templateSelectAllFunc)
	if err != nil {
		fmt.Println(templateSelectAllFunc)
		panic("Unable to parse template")
	}

	_, err = templates.New(`table`).Parse(templateTable)
	if err != nil {
		fmt.Println(templateTable)
		panic("Unable to parse template")
	}

	_, err = templates.New(`db`).Parse(templateDB)
	if err != nil {
		fmt.Println(string(templateDB))
		panic("Unable to parse template")
	}

	return templates
}
