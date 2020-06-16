package dice

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/BurntSushi/toml"
)

type modelData struct {
	BaseStmt     string
	Filter       string
	Dialect      DriverIdent
	ModelName    string
	Columns      string
	PK           string
	TableName    string
	FieldList    []colEquivalents
	initFileData string
}

type colEquivalents struct {
	ColName string
	Kind    reflect.Kind
	Attr    Structure
}

type compilerCache struct {
	// ColEquivalents provies a flat structure to
	// access the Field name of a table column
	// for example: map[posts.title]{ColName: "Title"}
	// will tell us that the data of column title
	// must be decoded inside Title field of our
	// dice.Model.
	ColEquivalents   map[string]colEquivalents
	Columns          map[string][]string
	ModelEquivalents map[string]string
}

// Compile parses your *.dice files and generates dice Models
// from `source` to dice.Model inside the `destination` path
// the model generated can depend upon the given engine.
// if all the compile logs needs to be displayed you can pass
// dice.Options{Verbose: true}
func Compile(source, destination string, opts Options) error {

	if opts.Verbose {
		setLogger(true)
	} else {
		setLogger(false)
	}

	err := checkConfig(source)
	if err != nil {
		return err
	}

	slog := log.Sugar()

	diceFiles, err := getDiceFiles(source)
	if err != nil {
		return err
	}

	if len(diceFiles) == 0 {
		fmt.Println("dice: nothing to compile")
		return nil
	}

	slog.Debugf("Found %d dice schema(s) in %s/ directory", len(diceFiles), source)
	slog.Debug("Analyzing dice schema(s)...")

	schemas, err := getSchemaList(diceFiles)
	if err != nil {
		return err
	}

	if len(schemas) == 0 {
		fmt.Println("dice: nothing to do")
		return nil
	}

	pk, cache, err := checkSchemas(schemas)
	if err != nil {
		return err
	}

	log.Sugar().Debugf("Primary keys are: %#v", pk)
	//log.Sugar().Debugf("Compiler cache generated: %#v", cache)

	cpath := getCachePath()
	p := encodeCompilerCache(cpath, cache)
	if p != "" {
		log.Sugar().Debugf("Compiler cache written to %s", p)
	} else {
		log.Sugar().Debug("Cannot write compiler cache. Problem specified above.")
	}

	if err != nil {
		return err
	}

	opts.Destination = destination
	generateModels(opts, pk, cache)

	return nil
}

// CompileCache only compiles the schama cache and saves it to
// the user home directory. It does not perform checks or any
// other strict checking and should be used only if migration
// didn't generate proper cache.
func CompileCache(source string, opts ...Options) error {
	if len(opts) > 0 && opts[0].Verbose {
		setLogger(true)
	} else {
		setLogger(false)
	}

	err := checkConfig(source)
	if err != nil {
		return err
	}

	slog := log.Sugar()

	slog.Debug("Compiling only cache...")

	diceFiles, err := getDiceFiles(source)
	if err != nil {
		return err
	}

	if len(diceFiles) == 0 {
		return fmt.Errorf("no dice files found under %s/", source)
	}

	schemas, err := getSchemaList(diceFiles)
	cache := compilerCache{
		ColEquivalents:   make(map[string]colEquivalents),
		Columns:          make(map[string][]string),
		ModelEquivalents: make(map[string]string),
	}

	for i := 0; i < len(schemas); i++ {
		s := schemas[i]
		cache.Columns[s.Table] = []string{}
		cache.ModelEquivalents[s.Table] = s.ModelName

		for cname, attr := range s.Columns {
			ceq := colEquivalents{}
			n := createStructName(cname)
			kind := getKindFromDiceConfig(attr.Type)
			ceq.ColName = n
			ceq.Kind = kind
			ceq.Attr = attr
			key := fmt.Sprintf("%s.%s", s.Table, cname)
			cache.ColEquivalents[key] = ceq

			cache.Columns[s.Table] = append(cache.Columns[s.Table], cname)
		}
	}

	cpath := getCachePath()
	p := encodeCompilerCache(cpath, cache)
	if p == "" {
		return errors.New("an error occured while writing dice cache file")
	}

	slog.Debugf("Cache written to %s", p)

	return nil
}

func checkSchemas(schemas []Schema) (map[string]string, compilerCache, error) {
	pk := make(map[string]string)
	cache := compilerCache{
		ColEquivalents:   make(map[string]colEquivalents),
		Columns:          make(map[string][]string),
		ModelEquivalents: make(map[string]string),
	}

	for i := 0; i < len(schemas); i++ {
		s := schemas[i]
		ceq := colEquivalents{}
		var cols []string

		// we verify if all columns have type variable in structure
		// and if there is not more than one primery key definitions
		// we also check if `using` attribute is used to define
		// a column then if the target column exists or not
		for cname, st := range s.Columns {
			if st.Type == "" {
				return pk,
					compilerCache{},
					fmt.Errorf("column for %s::%s doesn't have type field",
						s.ModelName, cname)
			}

			if st.TablePK && pk[s.Table] != "" {
				msg := "We already have a primary key %s, column %s" +
					" cannot be satisfied."
				return pk, compilerCache{}, fmt.Errorf(msg, pk, cname)
			} else if st.TablePK && pk[s.Table] == "" {
				pk[s.Table] = cname
			}

			if st.Using != "" && s.Columns[st.Using].Type == "" {
				msg := "defined using=\"%s\" for field: %s but %s" +
					" is not defined as a column"
				return pk, compilerCache{},
					fmt.Errorf(msg, st.Using, cname, st.Using)
			}

			n := createStructName(cname)
			kind := getKindFromDiceConfig(st.Type)
			ceq.ColName = n
			ceq.Kind = kind
			ceq.Attr = st
			key := fmt.Sprintf("%s.%s", s.Table, cname)
			cache.ColEquivalents[key] = ceq

			cols = append(cols, cname)
		}

		if pk[s.Table] == "" {
			return pk, cache,
				fmt.Errorf("table: %s does not have a primary key, not allowed",
					s.Table)

		}

		cache.Columns[s.Table] = cols
		cache.ModelEquivalents[s.Table] = s.ModelName

	}

	log.Sugar().Debug("Everything looks good, trying to create models...")

	return pk, cache, nil
}

func createStructName(column string) string {
	if !strings.Contains(column, "_") {
		c0 := column[0]
		return strings.ToUpper(string(c0)) + column[1:]
	}

	fin := ""
	foundUndie := false
	for i := 0; i < len(column); i++ {
		if i == 0 {
			fin += strings.ToUpper(string(column[i]))
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

func encodeCompilerCache(path string, cache compilerCache) string {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(cache)
	if err != nil {
		log.Sugar().Error(err)
		return ""
	}

	err = ioutil.WriteFile(path, buf.Bytes(), 0755)
	if err != nil {
		log.Sugar().Error(err)
		return ""
	}

	return path
}

func decodeCompilerCache(path string, cache *compilerCache) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.New("dice: compiler cache not found. try dice -cache" +
			" if migration is already done")

	}

	buf := bytes.NewBuffer(b)
	err = gob.NewDecoder(buf).Decode(cache)
	if err != nil {
		return err
	}

	return nil
}

func getDiceFiles(source string) ([]string, error) {
	var diceFiles []string
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".dice") {
			diceFiles = append(diceFiles, filepath.Join(source, info.Name()))
		}
		return err
	})

	return diceFiles, err
}

func getSchemaList(diceFiles []string) ([]Schema, error) {
	var schemas []Schema
	for _, p := range diceFiles {
		var s Schema
		_, err := toml.DecodeFile(p, &s)
		if err != nil {
			return schemas, err
		}

		//logToml(&s)

		if s.Table == "" {
			msg := "dice: table name for %s cannot be empty. add `table` to the root"
			return schemas, fmt.Errorf(msg, p)
		} else if s.ModelName == "" {
			msg := "dice: model name for %s cannot be empty. add `model` to the root"
			return schemas, fmt.Errorf(msg, p)
		} else if len(s.Columns) == 0 {
			msg := "dice: column list is empty for %s. Add [columns] object." +
				" not generating model\n"
			fmt.Printf(msg, p)
		} else {
			schemas = append(schemas, s)
		}
	}

	return schemas, nil
}

func generateModels(opts Options, pks map[string]string, cache compilerCache) {
	// clean the target models folder
	err := cleanDestinationFolder(opts.Destination)
	if err != nil {
		fmt.Println(err)
		return
	}

	for table, pk := range pks {
		md := modelData{}
		md.Dialect = opts.Dialect
		md.ModelName = cache.ModelEquivalents[table]
		md.TableName = table
		md.PK = pk
		var fl []colEquivalents
		var colFields []string
		for _, col := range cache.Columns[table] {
			key := fmt.Sprintf("%s.%s", table, col)
			fl = append(fl, cache.ColEquivalents[key])
			colFields = append(colFields, cache.ColEquivalents[key].ColName)
		}
		md.Columns = "\"" + strings.Join(colFields, "\",\"") + "\""
		md.FieldList = fl
		writeModelTemplate(md, opts.Destination)
	}
}

func checkConfig(source string) error {
	// check if config.toml is present to know the dialect
	confp := filepath.Join(source, "config.toml")
	if f, _ := os.Stat(confp); f == nil {
		return fmt.Errorf("config.toml not found under %s. cannot assert dialect, "+
			"do dice -init", source)
	}

	return nil
}

func getCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "dicecache.gob")
}

func getKindFromDiceConfig(ty string) reflect.Kind {
	switch ty {
	case "int":
		return reflect.Int
	case "string":
		return reflect.String
	default:
		return reflect.String
	}
}
