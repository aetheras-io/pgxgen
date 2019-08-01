package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Version = "0.1.0"

	FlagPGURL = "pgurl"
)

var (
	templates *template.Template
)

func init() {
	templates = loadTemplates()
}

func initCmd(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "init requires exactly one argument")
		os.Exit(1)
	}

	data := initData{
		PkgName: args[0],
		Version: Version,
	}

	err := os.Mkdir(data.PkgName, os.ModePerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	files := []struct {
		path string
		tmpl *template.Template
	}{
		{"config.toml", templates.Lookup("config")},
	}
	for _, f := range files {
		err := writeTemplateFile(filepath.Join(data.PkgName, f.path), f.tmpl, data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func writeTemplateFile(path string, tmpl *template.Template, data initData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func versionCmd(cmd *cobra.Command, args []string) {
	fmt.Println(Version)
	os.Exit(0)
}

func main() {
	cobra.EnableCommandSorting = false
	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("PGXG")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		viper.AutomaticEnv()
	})

	cmdInit := &cobra.Command{
		Use:   "init [PROJECT_NAME]",
		Short: "Initialize a new data package",
		Run:   initCmd,
	}

	cmdGenerate := &cobra.Command{
		Use:   "generate",
		Short: "Build",
		Run:   generateCmd,
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
			os.Exit(0)
		},
	}

	var rootCmd = &cobra.Command{
		Use: "pgxgen",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return viper.BindPFlags(cmd.Flags())
		},
	}
	rootCmd.PersistentFlags().String(FlagPGURL, "", "postgres url format")

	rootCmd.AddCommand(cmdInit)
	rootCmd.AddCommand(cmdGenerate)
	rootCmd.AddCommand(cmdVersion)
	rootCmd.Execute()
}
