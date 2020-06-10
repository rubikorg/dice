package dice

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
)

func setup() {
	spath := filepath.Join(".", "schemas")
	mpath := filepath.Join(".", "models")
	cfgp := filepath.Join(spath, "config.toml")
	dummysp := filepath.Join(spath, "posts.dice")
	dummySchema := `
table = "posts"
model = "Post"
create_dates = true

[columns]
id = { type = "int", table_pk = true }
title = { type = "string" }
comments = { type = "slice", model = "Comment", using = "comment_id" }
comment_id = { type = "int", constraint = "comments(id)" }
liked_count = { type = "int", ignore = true }
`
	os.MkdirAll(spath, 0755)
	os.MkdirAll(mpath, 0755)
	var buf bytes.Buffer
	toml.NewEncoder(&buf).Encode(Options{})
	ioutil.WriteFile(cfgp, buf.Bytes(), 0755)
	ioutil.WriteFile(dummysp, []byte(dummySchema), 0755)
}

func init() {
	setLogger(false)
	setup()
}

func TestGetDiceFiles(t *testing.T) {
	dfiles, err := getDiceFiles(filepath.Join(".", "schemas"))
	if err != nil {
		t.Error(err)
	}

	l := len(dfiles)
	if l == 0 {
		t.Errorf("getDiceFiles should return len 1 but got: %d", l)
	}
}

func TestGetSchemaList(t *testing.T) {
	dfiles, _ := getDiceFiles(filepath.Join(".", "schemas"))
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
	jdfPath := filepath.Join(".", "schemas", "json.dice")
	ioutil.WriteFile(jdfPath, []byte(jsonFile), 0755)
	dfiles, _ := getDiceFiles(filepath.Join(".", "schemas"))

	_, err := getSchemaList(dfiles)
	if err == nil {
		t.Error("getSchemaList() must throw error if it is reading JSON file. It did not")
	}

	os.Remove(jdfPath)
}

func TestCheckSchemas(t *testing.T) {
	dfiles, _ := getDiceFiles(filepath.Join(".", "schemas"))
	s, _ := getSchemaList(dfiles)

	pk, cache, err := checkSchemas(s)
	if err != nil {
		t.Error(err)
	}

	if len(pk) == 0 {
		t.Errorf("checkSchemas() 1st return value must have some values found %d values", len(pk))
	}

	if len(cache.ColEquivalents) == 0 {
		t.Errorf("checkSchemas() 2nd return value must have some values found %d values",
			len(cache.ColEquivalents))
	}
}

func TestCheckSchemas2(t *testing.T) {
	wrongSchema := `table = "w"
	model = "W"
	create_dates = true
	[columns]
	id = { table_pk = true }`
	sp := filepath.Join(".", "schemas")
	wsPath := filepath.Join(sp, "w.dice")
	ioutil.WriteFile(wsPath, []byte(wrongSchema), 0755)

	dfiles, _ := getDiceFiles(sp)
	s, _ := getSchemaList(dfiles)

	_, _, err := checkSchemas(s)
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
	corrp := filepath.Join(".", "schemas")

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
	testp := filepath.Join(".", "schemas", "testcache.gob")
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
