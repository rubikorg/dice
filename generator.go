package dice

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

var modelTemplate = `package models

import (
	"context"

	dice "github.com/rubikorg/dice"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type M{{ .ModelName }} struct {
	ID primitive.ObjectID ` + "`bson:\"_id\" json:\"_id\"`" + `
}

func (M{{ .ModelName }}) ColumnList() []string {
	return []string{ {{ .Columns }} }
}

func (M{{ .ModelName }}) PK() string {
	return "{{ .PK }}"
}

func {{ .ModelName }}sCollection() string {
	return "{{ .TableName }}"
}

func (g *M{{ .ModelName }}) FindOne(query dice.Q) {
	col := dice.GetDB().Collection({{ .ModelName }}sCollection())
	res := col.FindOne(context.TODO(), query)
	if res != nil {
		res.Decode(g)
	}
}

func (g *M{{ .ModelName }}) Save() primitive.ObjectID {
	col := dice.GetDB().Collection({{ .ModelName }}sCollection())
	if g.ID.IsZero() {
		g.ID = primitive.NewObjectID()
	}

	res, err := col.InsertOne(context.TODO(), *g)
	if err != nil {
		return primitive.ObjectID{}
	}

	return res.InsertedID.(primitive.ObjectID)
}

func (g *M{{ .ModelName }}) Delete() error {
	col := dice.GetDB().Collection({{ .ModelName }}sCollection())
	_, err := col.DeleteOne(context.TODO(), dice.Q{primitive.E{"_id", g.ID}})
	if err != nil {
		return err
	}

	return nil
}

func Find{{ .ModelName }}s(query dice.Q, g *[]M{{ .ModelName }}) error {
	col := dice.GetDB().Collection({{ .ModelName }}sCollection())
	cursor, err := col.Find(context.TODO(), query)
	if err != nil {
		return err
	}

	if err = cursor.All(context.TODO(), g); err != nil {
		return err
	}
	return nil
}

func Delete{{ .ModelName }}s(query dice.Q) error {
	col := dice.GetDB().Collection({{ .ModelName }}sCollection())
	_, err := col.DeleteMany(context.TODO(), query)
	if err != nil {
		return err
	}

	return nil
}

func {{ .ModelName }}() *M{{ .ModelName }} {
	var m M{{ .ModelName }}
	return  &m
}

func {{ .ModelName }}s() *[]M{{ .ModelName }} {
	var m []M{{ .ModelName }}
	return  &m
}
`

// TODO: make this a struct or an interface to serve for different dialect
var initTemplatePq = `package models

import (
	"io/ioutil"

	"github.com/rubikorg/dice"
	"github.com/rubikorg/dice/mgoconn"
	"gopkg.in/yaml.v2"
)

func Init() error {
	var opts dice.Options
	b, err := ioutil.ReadFile("./dice.yaml")
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, &opts)
	if err != nil {
		return err
	}

	db, err := mgoconn.Connect(opts.Credentials)
	if err != nil {
		return err
	}

	dice.Use(db, opts)
	return nil
}
`

func GenerateModel(name string) error {
	// TODO: make this generic accross dialect
	md := modelData{
		BaseStmt:  orm.opts.Base,
		Filter:    orm.opts.Filter,
		Dialect:   orm.opts.Dialect,
		ModelName: createStructName(name),
		TableName: name,
		PK:        "_id",
		Columns:   "",
	}

	modelPath := path.Join(".", "models")
	if err := desitinationChecks(name, modelPath); err != nil {
		return err
	}

	writeModelTemplate(md, modelPath)
	return nil
}

func writeModelTemplate(md modelData, dest string) {
	// determine what FilterStmt impl and BaseStmt needs to be used
	// extractDataFromModelCache(&md)

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

	buf.Reset()
	initTempl, err := template.New("init").Parse(initTemplatePq)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	err = initTempl.Execute(&buf, md)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(filepath.Join(dest, "init.go"), buf.Bytes(), 0755)
	if err != nil {
		fmt.Println(err)
	}

	fixedImports, err := imports.Process(modelPath, formatted, nil)
	// write formatted code to model file
	err = ioutil.WriteFile(modelPath, fixedImports, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func desitinationChecks(name, dest string) error {
	if f, _ := os.Stat(dest); f == nil {
		if err := os.MkdirAll(dest, 0755); err != nil {
			return err
		}
	}

	modelPath := filepath.Join(dest, name+".go")
	if f, _ := os.Stat(modelPath); f != nil {
		return fmt.Errorf("model: %s already exists", name)
	}

	return nil
}

// TODO: remove this once confirmed we would not need this ever
// func extractDataFromModelCache(md *modelData) {
// 	switch md.Dialect {
// 	// TODO: change this to proper driver connection later
// 	case Postgres, MySQL, SQLite:
// 		md.BaseStmt = "PqBase"
// 		md.Filter = "SQLFilter"
// 		md.initFileData = initTemplatePq
// 	default:
// 		md.BaseStmt = "PqBase"
// 		md.Filter = "SQLFilter"
// 		md.initFileData = initTemplatePq
// 	}
// }

func createStructName(column string) string {
	name := column
	if !strings.Contains(name, "_") {
		c0 := name[0]
		if name[len(name)-1] == 's' {
			name = column[:len(name)-1]
		}
		return strings.ToUpper(string(c0)) + name[1:]
	}

	fin := ""
	foundUndie := false
	for i := 0; i < len(column); i++ {
		if i == 0 {
			fin += strings.ToUpper(string(column[i]))
			continue
		}

		if i == len(column)-1 && column[i] == 's' {
			continue
		}

		if column[i] == '_' {
			foundUndie = true
			continue
		}

		if foundUndie {
			fin += strings.ToUpper(string(column[i]))
			foundUndie = false
		} else {
			fin += string(column[i])
		}

	}
	return fin
}
