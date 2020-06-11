package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/rubikorg/dice"
)

var (
	src     string
	dest    string
	migrate bool
	cache   bool
	initF   bool
	clean   bool
	srcp    = filepath.Join(".", "schemas")
	destp   = filepath.Join(".", "models")
)

func main() {
	flag.StringVar(&src, "src", "", "source of your dice schema definitions")
	flag.StringVar(&dest, "dest", "", "the destination of your compiled Go models")

	flag.BoolVar(&migrate, "migrate", false, "run the migration with given -src and -dest")
	flag.BoolVar(&cache, "cache", false, "generate new dice compiler cache")
	flag.BoolVar(&initF, "init", false, "initialize a config.toml for compiler configuration")
	flag.BoolVar(&clean, "clean", false, "cleans the models and compiler cache")

	flag.Parse()

	if len(flag.Args()) > 0 && flag.Args()[0] == "help" {
		fmt.Print("Dice command line help screen: \n\n")
		flag.PrintDefaults()
		return
	}

	conf := getDiceOpts()
	if conf.Source == "" || conf.Destination == "" {
		conf.Source = srcp
		conf.Destination = destp
	}

	if clean {
		err := cleanFlagAction(conf)
		if err != nil {
			log.Fatal(err)
		}
	}

	// if init flag is given but there is no custom src
	// folders provided write to default ones
	if initF && src == "" {
		os.MkdirAll(srcp, 0755)
		err := writeNewConfig(srcp)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("generated config.toml inside ./%s. please change the dialect according to"+
			" your database", srcp)

		return
	} else if initF && src != "" {
		os.MkdirAll(src, 0755)
		err := writeNewConfig(src)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	// same goes for -migrate flag if -src and -dest is not provided
	// write to srcp and destp variables
	if migrate && src == "" && dest == "" {
		os.MkdirAll(srcp, 0755)
		os.MkdirAll(destp, 0755)
		err := dice.Compile(srcp, destp, dice.Postgres, dice.Options{Verbose: true})
		if err != nil {
			panic(err)
		}
		return
	} else if migrate && src != "" && dest != "" {
		os.MkdirAll(src, 0755)
		os.MkdirAll(dest, 0755)
		err := dice.Compile(src, dest, dice.Postgres, dice.Options{Verbose: true})
		if err != nil {
			panic(err)
		}
		return
	}

	conf.Verbose = true

	// same logic as -migrate flag .. compile only cache
	if cache && conf.Source == "" {
		err := dice.CompileCache(srcp, conf)
		if err != nil {
			panic(err)
		}
	} else if cache && conf.Source != "" {
		err := dice.CompileCache(conf.Source, conf)
		if err != nil {
			panic(err)
		}
	}
}

func cleanFlagAction(conf dice.Options) error {
	err := filepath.Walk(conf.Destination,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				os.Remove(filepath.Join(conf.Destination, info.Name()))
			}

			return err
		})

	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	os.Remove(filepath.Join(home, "dicecache.gob"))

	return err
}

func getDiceOpts() dice.Options {
	var conf dice.Options
	if src != "" {
		confp := filepath.Join(src, "config.toml")
		_, err := toml.DecodeFile(confp, &conf)
		if err != nil {
			panic(err)
		}
	} else {
		confp := filepath.Join(srcp, "config.toml")
		if f, _ := os.Stat(confp); f != nil {
			_, err := toml.DecodeFile(confp, &conf)
			if err != nil {
				panic(err)
			}
		}
	}

	return conf
}

// writeNewConfig writes a new dice.Options{} into the src
// directory that you provide inside config.toml file.
func writeNewConfig(src string) error {
	opts := dice.Options{}
	opts.Dialect = string(dice.Postgres)

	confp := filepath.Join(src, "config.toml")
	if f, _ := os.Stat(confp); f != nil {
		return nil
	}

	var buf bytes.Buffer
	err := toml.NewEncoder(&buf).Encode(&opts)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(confp, buf.Bytes(), 0755)
	if err != nil {
		return err
	}

	return nil
}
