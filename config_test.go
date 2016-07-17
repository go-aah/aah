// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/config source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package config

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

func initString(t *testing.T, configStr string) *Config {
	cfg, err := ParseString(configStr)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	return cfg
}

func initFile(t *testing.T, file string) *Config {
	cfg, err := LoadFile(file)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	return cfg
}

func SetProfile(t *testing.T, cfg *Config, profile string) {
	err := cfg.SetProfile(profile)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func TestStringValues(t *testing.T) {
	cfg := initFile(t, "testdata/test.cfg")

	v1, _ := cfg.String("string")
	assertEqual(t, "TestStringValues", "a string", v1)
	assertEqual(t, "TestStringValues", cfg.StringDefault("string_not_exists", "nice 1"), "nice 1")
	assertEqual(t, "TestStringValues", cfg.StringDefault("string", "nice 1"), "a string")

	SetProfile(t, cfg, "dev")
	dv1, _ := cfg.String("string")
	assertEqual(t, "TestStringValues - dev", "a string inside dev", dv1)
	assertEqual(t, "TestStringValues - dev", cfg.StringDefault("string_not_exists", "nice 2"), "nice 2")

	SetProfile(t, cfg, "prod")
	pv1, _ := cfg.String("string")
	assertEqual(t, "TestStringValues - prod", "a string inside prod", pv1)
	assertEqual(t, "TestStringValues - prod", cfg.StringDefault("string_not_exists", "nice 3"), "nice 3")
}

func TestIntValues(t *testing.T) {
	bytes, _ := ioutil.ReadFile("testdata/test.cfg")
	cfg := initString(t, string(bytes))

	v1, _ := cfg.Int("int")
	assertEqual(t, "TestIntValues", 32, v1)
	v2, _ := cfg.Int64("int64")
	assertEqual(t, "TestIntValues", int64(1), v2)
	v3, _ := cfg.Int64("int64not")
	assertEqual(t, "TestIntValues", int64(0), v3)
	assertEqual(t, "TestIntValues", cfg.IntDefault("int_not_exists", 99), 99)
	assertEqual(t, "TestIntValues", cfg.IntDefault("int", 99), 32)

	SetProfile(t, cfg, "dev")
	dv1, _ := cfg.Int("int")
	assertEqual(t, "TestIntValues - dev", 500, dv1)
	dv2, _ := cfg.Int64("int64")
	assertEqual(t, "TestIntValues - dev", int64(2), dv2)
	assertEqual(t, "TestIntValues - dev", cfg.IntDefault("int_not_exists", 199), 199)

	SetProfile(t, cfg, "prod")
	pv1, _ := cfg.Int("int")
	assertEqual(t, "TestIntValues - prod", 1000, pv1)
	pv2, _ := cfg.Int64("int64")
	assertEqual(t, "TestIntValues - prod", int64(3), pv2)
	assertEqual(t, "TestIntValues - prod", cfg.IntDefault("int_not_exists", 299), 299)
}

func TestFloatValues(t *testing.T) {
	cfg := initFile(t, "testdata/test.cfg")

	v1, _ := cfg.Float32("float32")
	assertEqual(t, "TestFloatValues", float32(32.2), v1)
	v2, _ := cfg.Float64("float64")
	assertEqual(t, "TestFloatValues", float64(1.1), v2)
	v3, _ := cfg.Float64("subsection.sub_float")
	assertEqual(t, "TestFloatValues", float64(10.5), v3)
	v4, _ := cfg.Float64("float64not")
	assertEqual(t, "TestFloatValues", float64(0.0), v4)
	assertEqual(t, "TestFloatValues", cfg.Float32Default("float_not_exists", float32(99.99)), float32(99.99))
	assertEqual(t, "TestFloatValues", cfg.Float32Default("float32", float32(99.99)), float32(32.2))

	SetProfile(t, cfg, "dev")
	dv1, _ := cfg.Float32("float32")
	assertEqual(t, "TestFloatValues - dev", float32(62.2), dv1)
	dv2, _ := cfg.Float64("float64")
	assertEqual(t, "TestFloatValues - dev", float64(2.1), dv2)
	dv3, _ := cfg.Float64("subsection.sub_float")
	assertEqual(t, "TestFloatValues - dev", float64(50.5), dv3)
	assertEqual(t, "TestFloatValues - dev", cfg.Float32Default("float_not_exists", float32(199.99)), float32(199.99))

	SetProfile(t, cfg, "prod")
	pv1, _ := cfg.Float32("float32")
	assertEqual(t, "TestFloatValues - prod", float32(122.2), pv1)
	pv2, _ := cfg.Float64("float64")
	assertEqual(t, "TestFloatValues - prod", float64(3.1), pv2)
	pv3, _ := cfg.Float64("subsection.sub_float")
	assertEqual(t, "TestFloatValues - prod", float64(100.5), pv3)
	assertEqual(t, "TestFloatValues - prod", cfg.Float32Default("float_not_exists", float32(299.99)), float32(299.99))
}

func TestBoolValues(t *testing.T) {
	bytes, _ := ioutil.ReadFile("testdata/test.cfg")
	cfg := initString(t, string(bytes))

	v1, _ := cfg.Bool("truevalue")
	assertEqual(t, "TestBoolValues", true, v1)
	v2, _ := cfg.Bool("falsevalue")
	assertEqual(t, "TestBoolValues", false, v2)
	assertEqual(t, "TestBoolValues", cfg.BoolDefault("bool_not_exists", true), true)
	assertEqual(t, "TestBoolValues", cfg.BoolDefault("falsevalue", true), false)

	SetProfile(t, cfg, "dev")
	assertEqual(t, "TestBoolValues - dev", cfg.BoolDefault("truevalue", true), true)    // keys is found by fallback
	assertEqual(t, "TestBoolValues - dev", cfg.BoolDefault("falsevalue", false), false) // keys is found by fallback

	SetProfile(t, cfg, "prod")
	pv1, _ := cfg.Bool("truevalue")
	assertEqual(t, "TestBoolValues - prod", true, pv1)
	pv2, _ := cfg.Bool("falsevalue")
	assertEqual(t, "TestBoolValues - prod", false, pv2)
	assertEqual(t, "TestBoolValues - prod", cfg.BoolDefault("bool_not_exists", true), true)
}

func TestProfile(t *testing.T) {
	cfg := initFile(t, "testdata/test.cfg")

	t.Log(cfg.cfg.Keys())

	assertEqual(t, "TestProfile", "", cfg.Profile())
	assertEqual(t, "TestProfile",
		"profile doesn't exists: not_exists_profile",
		cfg.SetProfile("not_exists_profile").Error())

	cfg.ClearProfile()
	assertEqual(t, "TestProfile", true, cfg.profile == "")
}

func TestConfigLoadNotExists(t *testing.T) {
	_, err := LoadFile("testdata/not_exists.cfg")
	assertEqual(t, "TestConfigLoadNotExists - 1",
		"open testdata/not_exists.cfg: no such file or directory",
		err.Error())

	_, err = ParseString(`
  # Error configuration
  string = "a string"
  int = 32 # adding comment without semicolon will lead to error
  float32 = 32.2
  int64 = 1
  float64 = 1.1
    `)
	assertEqual(t, "TestConfigLoadNotExists - 2", true,
		strings.Contains(err.Error(), "adding comment without semicolon will lead to error"))
}

func assertEqual(t *testing.T, method, e, g interface{}) (r bool) {
	r = compare(e, g)
	if !r {
		t.Errorf("%v: Expected [%v], got [%v]", method, e, g)
	}

	return
}

func TestMergeConfig(t *testing.T) {
	cfg1, err := ParseString(`
global = "global value";

prod {
	value = "string value";
	integer = 500
	float = 80.80
	boolean = true
	negative = FALSE
	nothing = NULL
}
	`)

	cfg2, err := ParseString(`
global = "global value";

newvalue = "I'm new value"
prod {
  value = "I'm prod value"
	nothing = 200
}
`)

	err = cfg1.Merge(cfg2)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	_ = cfg1.SetProfile("prod")

	v1 := cfg1.IntDefault("nothing", 0)
	assertEqual(t, "TestMergeConfig", 200, v1)

	v2 := cfg1.StringDefault("value", "")
	assertEqual(t, "TestMergeConfig", "I'm prod value", v2)

	v3 := cfg1.StringDefault("newvalue", "")
	assertEqual(t, "TestMergeConfig", "I'm new value", v3)

	err = cfg1.Merge(nil)
	assertEqual(t, "TestMergeConfig", "source is nil", err.Error())
}

func TestLoadFiles(t *testing.T) {
	cfg, err := LoadFiles("testdata/test-1.cfg", "testdata/test-2.cfg", "testdata/test-3.cfg")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assertEqual(t, "TestLoadFiles", float32(10.5), cfg.Float32Default("subsection.sub_float", 0.0))
	assertEqual(t, "TestLoadFiles", float32(32.4), cfg.Float32Default("float32", 0.0))

	_ = cfg.SetProfile("dev")
	assertEqual(t, "TestLoadFiles - dev", "a string inside dev from test-2", cfg.StringDefault("string", ""))
	assertEqual(t, "TestLoadFiles - dev", float32(500.5), cfg.Float32Default("subsection.sub_float", 0.0))
	assertEqual(t, "TestLoadFiles - dev", float32(62.2), cfg.Float32Default("float32", 0.0))

	_ = cfg.SetProfile("prod")
	assertEqual(t, "TestLoadFiles - prod", "a string inside prod from test-3", cfg.StringDefault("string", ""))
	assertEqual(t, "TestLoadFiles - prod", float32(1000.5), cfg.Float32Default("subsection.sub_float", 0.0))
	assertEqual(t, "TestLoadFiles - prod", float32(222.2), cfg.Float32Default("float32", 0.0))
	assertEqual(t, "TestLoadFiles - prod", true, cfg.BoolDefault("falsevalue", false))
	assertEqual(t, "TestLoadFiles - prod", false, cfg.BoolDefault("truevalue", true))

	// fail cases
	_, err = LoadFiles("testdata/not_exists.cfg")
	assertEqual(t, "TestLoadFiles - not exists", true, strings.Contains(err.Error(), "no such file or directory"))

	_, err = LoadFiles("testdata/test-1.cfg", "testdata/test-error.cfg")
	assertEqual(t, "TestLoadFiles - merge error", true, strings.Contains(err.Error(), "source (STRING) and target (SECTION)"))
}

func compare(e, g interface{}) (r bool) {
	ev := reflect.ValueOf(e)
	gv := reflect.ValueOf(g)

	if ev.Kind() != gv.Kind() {
		return
	}

	switch ev.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		r = (ev.Int() == gv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		r = (ev.Uint() == gv.Uint())
	case reflect.Float32, reflect.Float64:
		r = (ev.Float() == gv.Float())
	case reflect.String:
		r = (ev.String() == gv.String())
	case reflect.Bool:
		r = (ev.Bool() == gv.Bool())
	}

	return
}
