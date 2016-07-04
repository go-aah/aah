// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package config

import (
	"fmt"
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

	fmt.Println(cfg.cfg.Keys())

	assertEqual(t, "TestProfile", "", cfg.Profile())
	assertEqual(t, "TestProfile",
		"profile doesn't exists: not_exists_profile",
		cfg.SetProfile("not_exists_profile").Error())

	cfg.ClearProfile()
	assertEqual(t, "TestProfile", true, cfg.profile == "")
}

func TestConfigLoadNotExists(t *testing.T) {
	_, err := LoadFile("testdata/not_exists.cfg")
	assertEqual(t, "TestConfigLoadNotExists",
		"open testdata/not_exists.cfg: no such file or directory",
		err.Error())

	_, err = ParseString(`
  # Error configuration
  string = "a string"
  int = 32 # adding comment with semicolon will lead to error
  float32 = 32.2
  int64 = 1
  float64 = 1.1
    `)
	assertEqual(t, "method", true,
		strings.Contains(err.Error(), "adding comment with semicolon will lead to error"))
}

func assertEqual(t *testing.T, method, e, g interface{}) (r bool) {
	r = compare(e, g)
	if !r {
		t.Errorf("%v: Expected [%v], got [%v]", method, e, g)
	}

	return
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
