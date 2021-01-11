package main

// import (
// 	"io/ioutil"
// 	"os"
// 	"path/filepath"
// 	"testing"

// 	"github.com/rubikorg/dice"
// )

// var tsrcp string

// func init() {
// 	os.Setenv("devenv", "test")
// 	tsrcp = filepath.Join("..", "..", "schemas_test")
// }

// func TestCleanFlagAction(t *testing.T) {
// 	tdestp := filepath.Join("..", "..", "models")
// 	conf := dice.Options{
// 		Source:      tsrcp,
// 		Destination: tdestp,
// 	}
// 	filep := filepath.Join(tdestp, "file.go")
// 	ioutil.WriteFile(filep, []byte("hello"), 0755)

// 	err := cleanFlagAction(conf)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if f, _ := os.Stat(filep); f != nil {
// 		t.Errorf("cleanFlagAction() did not clean %s", filep)
// 	}

// 	conf.Destination = filepath.Join(".", "models")
// 	err = cleanFlagAction(conf)
// 	if err == nil {
// 		t.Error("cleanFlagAction() did not return error for wrong destination")
// 	}
// }

// func TestGetDiceOpts(t *testing.T) {
// 	src = tsrcp
// 	opts := getDiceOpts()
// 	if opts.Dialect != "postgres" {
// 		t.Error("getDiceOpts() did not parse config.yaml properly")
// 	}

// 	newOpts := getDiceOpts()
// 	if newOpts.Dialect != "postgres" {
// 		t.Error("getDiceOpts() did not parse config.yaml properly on flag passed")
// 	}
// }

// func TestWriteNewConfig(t *testing.T) {
// 	err := writeNewConfig(srcp)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	cf := filepath.Join(srcp, "config.yaml")
// 	if f, _ := os.Stat(cf); f == nil {
// 		t.Errorf("writeNewConfig() did not write the file inside %s", cf)
// 	}
// }

// func TestInitCacheFlag(t *testing.T) {
// 	cache = true
// 	src = tsrcp
// 	dest = filepath.Join("..", "..", "models")
// 	main()

// 	home, _ := os.UserHomeDir()
// 	cachep := filepath.Join(home, "dicecache.gob")
// 	if f, _ := os.Stat(cachep); f == nil {
// 		t.Errorf("-cache did not create cache file inside %s", cachep)
// 	}
// }
