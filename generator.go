package dice

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"golang.org/x/tools/imports"
)

var modelTemplate = `package models

import dice "github.com/rubikorg/dice"

type M{{ .ModelName }} struct {
	{{- range .FieldList }}
	{{ .ColName }} {{ if eq .Attr.Type "slice" }} []string {{ else }} {{ .Attr.Type -}} {{ end }}
	{{- end }}
}

func (M{{ .ModelName }}) ColumnList() []string {
	return []string{ {{ .Columns }} }
}

func (M{{ .ModelName }}) PK() string {
	return "{{ .PK }}"
}

func (M{{ .ModelName }}) TableName() string {
	return "{{ .TableName }}"
}

func {{ .ModelName }}() (*M{{ .ModelName }}, dice.BaseStmt) {
	var m M{{ .ModelName }}
	return &m, dice.{{ .BaseStmt }}{}.Target(&m)
}

func {{ .ModelName }}s() (*[]M{{ .ModelName }}, dice.BaseStmt) {
	var m []M{{ .ModelName }}
	return &m, dice.{{ .BaseStmt }}{}.Target(&m)
}
`

var initTemplatePq = `package models

import (
	"github.com/rubikorg/dice"
	"github.com/rubikorg/dice/postgres"
	"github.com/BurntSushi/toml"
)

func Init() error {
	var opts dice.Options
	_, err := toml.DecodeFile("./dice.toml", &opts)
	db, err := postgres.Connect(opts.Credentials)
	if err != nil {
		return err
	}

	dice.Use(opts.Dialect, db, opts)
	return nil
}
`

func writeModelTemplate(md modelData, dest string) {
	// determine what FilterStmt impl and BaseStmt needs to be used
	extractDataFromModelCache(&md)

	var buf bytes.Buffer
	fileName := fmt.Sprintf("%s.go", md.TableName)
	modelPath := filepath.Join(dest, fileName)

	templ, err := template.New("model").Parse(modelTemplate)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = templ.Execute(&buf, md)
	if err != nil {
		fmt.Println(err)
		return
	}

	// run gofmt on this generated model source code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Println(err)
		return
	}

	// TODO: think about where to put this init code
	// err = ioutil.WriteFile(filepath.Join(dest, "init.go"), []byte(md.initFileData), 0755)
	// if err != nil {
	// 	fmt.Println(err)
	// }

	fixedImports, err := imports.Process(modelPath, formatted, nil)
	// write formatted code to model file
	err = ioutil.WriteFile(modelPath, fixedImports, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func cleanDestinationFolder(dest string) error {
	dir, err := ioutil.ReadDir(dest)
	for _, d := range dir {
		os.RemoveAll(filepath.Join([]string{dest, d.Name()}...))
	}

	if err != nil {
		return err
	}
	return nil
}

func extractDataFromModelCache(md *modelData) {
	switch md.Dialect {
	// TODO: change this to proper driver connection later
	case Postgres, MySQL, SQLite:
		md.BaseStmt = "PqBase"
		md.Filter = "SQLFilter"
		md.initFileData = initTemplatePq
	default:
		md.BaseStmt = "PqBase"
		md.Filter = "SQLFilter"
		md.initFileData = initTemplatePq
	}
}
