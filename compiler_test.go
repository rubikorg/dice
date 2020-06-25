package dice

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

var spath string

func setup() {
	spath = filepath.Join(".", "schemas_test")
	mpath := filepath.Join(".", "models")
	tpath := filepath.Join(".", "test")
	cfgp := filepath.Join(spath, "config.yaml")
	dummysp := filepath.Join(spath, "posts.dice")
	dummySchema := `
table: "posts"
model: "Post"
create_dates: true

columns:
  id: { type: "int", table_pk: true }
  title: { type: "string" }
  comments: { type: "slice", model: "Comment", using: "comment_id" }
  comment_id: { type: "int", constraint: "comments(id)" }
  liked_count: { type: "int", ignore: true }`

	os.MkdirAll(spath, 0755)
	os.MkdirAll(mpath, 0755)
	os.MkdirAll(tpath, 0755)

	if cf, _ := os.Stat(cfgp); cf == nil {
		b, _ := yaml.Marshal(Options{Dialect: "postgres"})
		ioutil.WriteFile(cfgp, b, 0755)
	}

	if dsf, _ := os.Stat(dummysp); dsf == nil {
		ioutil.WriteFile(dummysp, []byte(dummySchema), 0755)
	}
}

func init() {
	setLogger(false)
	setup()
}

func TestGetDiceFiles(t *testing.T) {
	dfiles, err := getDiceFiles(filepath.Join(spath))
	if err != nil {
		t.Error(err)
	}

	l := len(dfiles)
	if l == 0 {
		t.Errorf("getDiceFiles should return len 1 but got: %d", l)
	}
}

func TestGetSchemaList(t *testing.T) {
	dfiles, _ := getDiceFiles(spath)
	l := len(dfiles)
	if l == 0 {
		t.Errorf("getDiceFiles() should return len 1 but got: %d", l)
	}

	s, err := getSchemaList(dfiles)
	if err != nil {
		t.Error(err)
	}

	sl := len(s)
	if sl == 0 {
		t.Errorf("getSchemaList() should return len 1 but got: %d", sl)
	}

	if s[0].Table != "posts" {
		t.Errorf("getSchemaList() did not parse the schema properly %v", s)
	}
}

func TestGetSchemaList2(t *testing.T) {
	// we expect an error if toml is not parsable so the best way
	// is to feed it a JSON ..lol
	jsonFile := `{"hello": "world"}`
	jdfPath := filepath.Join(spath, "json.dice")
	ioutil.WriteFile(jdfPath, []byte(jsonFile), 0755)
	dfiles, _ := getDiceFiles(filepath.Join(spath))

	_, err := getSchemaList(dfiles)
	if err == nil {
		t.Error("getSchemaList() must throw error if it is reading JSON file. It did not")
	}

	os.Remove(jdfPath)
}

func TestCheckSchemas(t *testing.T) {
	dfiles, _ := getDiceFiles(filepath.Join(spath))
	s, _ := getSchemaList(dfiles)

	err := checkSchemas(s)
	if err != nil {
		t.Error(err)
	}

	// TODO: abstract this part into a new test for getModelProperties

	// if len(pk) == 0 {
	// 	t.Errorf("checkSchemas() 1st return value must have some values found %d values", len(pk))
	// }
	//
	// if len(cache.ColEquivalents) == 0 {
	// 	t.Errorf("checkSchemas() 2nd return value must have some values found %d values",
	// 		len(cache.ColEquivalents))
	// }
}

func TestCheckSchemas2(t *testing.T) {
	wrongSchema := `
table: "w"
model: "W"
create_dates: true
columns:
  id: { table_pk: true }`
	sp := filepath.Join(spath)
	wsPath := filepath.Join(sp, "w.dice")
	ioutil.WriteFile(wsPath, []byte(wrongSchema), 0755)

	dfiles, _ := getDiceFiles(sp)
	s, _ := getSchemaList(dfiles)

	err := checkSchemas(s)
	if err == nil {
		t.Error("checkSchemas() did not error when type is not defined in column `id`")
	}

	os.Remove(wsPath)
}

func TestEncodeCompilerCache(t *testing.T) {
	testp := filepath.Join(".", "test", "dicecache.gob")
	p := encodeCompilerCache(testp, compilerCache{})

	if f, _ := os.Stat(testp); f == nil {
		t.Errorf("encodeCompilerCache() did not write cache to %s", testp)
	}

	if p != testp {
		t.Errorf("encodeCompilerCache() did not written the path back; returned %s", p)
	}

	os.Remove(testp)
}

func TestGetCachePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, "dicecache.gob")
	cp := getCachePath()
	if p != cp {
		t.Errorf("getCachePath() did not return the correct path: %s", cp)
	}
}

func TestCheckConfig(t *testing.T) {
	wrongp := filepath.Join(".", "meh")
	corrp := filepath.Join(spath)

	werr := checkConfig(wrongp)
	if werr == nil {
		t.Error("checkConfig() did not return an error for wrong source path")
	}

	cerr := checkConfig(corrp)
	if cerr != nil {
		t.Errorf("checkConfig() returned an error for the right source path: \n\n%s",
			cerr.Error())
	}
}

func TestDecodeCompilerCache(t *testing.T) {
	cache := compilerCache{}
	testp := filepath.Join(spath, "testcache.gob")
	p := encodeCompilerCache(testp, cache)
	if p == "" {
		t.Error("encodeCompilerCache() errored while testing decompile")
	}

	var dcache compilerCache
	if f, _ := os.Stat(testp); f != nil {
		err := decodeCompilerCache(testp, &dcache)
		if err != nil {
			t.Error(err)
		}
	} else {
		t.Errorf("encoded file not found in %s", testp)
	}

	os.Remove(testp)

	err := decodeCompilerCache(testp, &dcache)
	if err == nil {
		t.Error("decodeCompilerCache() should throw error when it did not find cache file")
	}
}

func TestGetSchemaListRequiredData(t *testing.T) {
	wrongSchema := Schema{}
	wrongsp := filepath.Join(spath, "user.dice")
	var buf bytes.Buffer
	yaml.NewEncoder(&buf).Encode(wrongSchema)
	ioutil.WriteFile(wrongsp, buf.Bytes(), 0755)

	df, _ := getDiceFiles(filepath.Dir(wrongsp))
	_, err := getSchemaList(df)
	if err == nil {
		t.Error("getSchemaList() did not return error if table name is not mentioned")
	}

	wrongSchema.Table = "users"
	testWriteSchema(wrongsp, &buf, wrongSchema)
	_, err = getSchemaList(df)
	if err == nil {
		t.Error("getSchemaList() did not return error if model name is not mentioned")
	}

	wrongSchema.ModelName = "User"
	testWriteSchema(wrongsp, &buf, wrongSchema)
	sch, _ := getSchemaList(df)
	// this means that the shchema was added for generation of models eventhough
	// there is no columns
	if len(sch) > 1 {
		t.Errorf("getSchemaList() returned more than 1 model [when] only 1 is correct %v", sch)
	}

	os.Remove(wrongsp)
}

func TestCompileCache(t *testing.T) {
	cachep := getCachePath()
	os.Remove(cachep)
	srcp := filepath.Join(spath)
	err := CompileCache(srcp)
	if err != nil {
		t.Error(err)
	}

	if f, _ := os.Stat(cachep); f == nil {
		t.Errorf("CompileCache() did not generate cache at %s even when correct schema", cachep)
	}
}

func TestCompile(t *testing.T) {
	srcp := filepath.Join(spath)
	destp := filepath.Join(".", "models")
	opts := Options{Verbose: false, Dialect: Postgres}
	err := Compile(srcp, destp, opts)
	if err != nil {
		t.Error(err)
	}
}

func testWriteSchema(path string, buf *bytes.Buffer, s Schema) {
	buf.Reset()
	yaml.NewEncoder(buf).Encode(s)
	ioutil.WriteFile(path, buf.Bytes(), 0755)
}
