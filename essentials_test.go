// Copyright (c) 2016 Jeevanandam M (https://github.com/jeevatkm)
// resty source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"os"
	"reflect"
	"testing"
)

func failOnError(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
}

func removeFiles(files ...string) {
	for _, f := range files {
		_ = os.Remove(f)
	}
}

func removeAllFiles(files ...string) {
	for _, f := range files {
		_ = os.RemoveAll(f)
	}
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
