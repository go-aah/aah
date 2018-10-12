// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"html/template"
	"io"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"aahframe.work/ahttp"
	"aahframe.work/log"
)

var (
	// ErrTypeOrParserIsNil returned when supplied `reflect.Type` or parser is nil to
	// the method `AddValueParser`.
	ErrTypeOrParserIsNil = errors.New("valpar: type or value parser is nil")

	// ErrValueParserIsAlreadyExists returned when given `reflect.Type` is already exists
	// in type parser list.
	ErrValueParserIsAlreadyExists = errors.New("valpar: value parser is already exists")

	// TimeFormats is configured values from aah.conf under `format { ... }`
	TimeFormats []string

	// StructTagName is used while binding struct fields.
	StructTagName string

	kindHandlers = map[reflect.Kind]Parser{
		reflect.Int:     handleTypes,
		reflect.Int8:    handleTypes,
		reflect.Int16:   handleTypes,
		reflect.Int32:   handleTypes,
		reflect.Int64:   handleTypes,
		reflect.Uint:    handleTypes,
		reflect.Uint8:   handleTypes,
		reflect.Uint16:  handleTypes,
		reflect.Uint32:  handleTypes,
		reflect.Uint64:  handleTypes,
		reflect.Float32: handleTypes,
		reflect.Float64: handleTypes,
		reflect.String:  handleTypes,
		reflect.Bool:    handleTypes,
		reflect.Slice:   handleSlice,
	}

	typeParsers = map[reflect.Type]Parser{
		timeType: handleTypes,
	}

	timeType = reflect.TypeOf(time.Time{})
)

// Parser interface is used to implement string -> type value parsing. This is
// similar to standard `strconv` package. It deals with reflect value.
type Parser func(key string, typ reflect.Type, params url.Values) (reflect.Value, error)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// AddValueParser method adds given custom value parser for the `reflect.Type`
func AddValueParser(typ reflect.Type, parser Parser) error {
	if typ == nil || parser == nil {
		return ErrTypeOrParserIsNil
	}

	if _, found := typeParsers[typ]; found {
		return ErrValueParserIsAlreadyExists
	}

	typeParsers[typ] = parser
	return nil
}

// ValueParser method returns the parser based on `reflect.Type` and `reflect.Kind`.
// It returns most of the value parser except Pointer and Struct kind.
// Since Pointer and Struct handled separately.
func ValueParser(typ reflect.Type) (Parser, bool) {
	typ, _ = checkPtr(typ)
	if parserFn, found := typeParsers[typ]; found {
		return parserFn, found
	} else if parserFn, found := kindHandlers[typ.Kind()]; found {
		return parserFn, found
	}
	return nil, false
}

// Body method parse the body based on Content-Type.
func Body(contentType string, body io.Reader, typ reflect.Type) (reflect.Value, error) {
	var err error
	s := reflect.New(typ)
	switch contentType {
	case ahttp.ContentTypeJSON.Mime, ahttp.ContentTypeJSONText.Mime:
		if err = json.NewDecoder(body).Decode(s.Interface()); err != nil {
			log.Errorf("json: %s", err)
			return s.Elem(), err
		}
	case ahttp.ContentTypeXML.Mime, ahttp.ContentTypeXMLText.Mime:
		if err = xml.NewDecoder(body).Decode(s.Interface()); err != nil {
			log.Error(err)
			return s.Elem(), err
		}
	}

	return s.Elem(), nil
}

// Struct method parses the value based on Content-Type. It handles JSON and XML.
func Struct(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
	var err error
	var isPtr bool
	typ, isPtr = checkPtr(typ)
	s := reflect.New(typ)
	st := s.Type().Elem()
	sv := s.Elem()

	for idx := 0; idx < st.NumField(); idx++ {
		ft := st.Field(idx)
		f := sv.Field(idx)
		if !f.CanSet() {
			continue
		}

		fname := ft.Tag.Get(StructTagName)
		if fname == "-" { // skip the field
			continue
		}

		if len(key) > 0 {
			fname = key + "." + fname
		}

		var v reflect.Value
		if vpFn, found := ValueParser(f.Type()); found {
			v, err = vpFn(fname, f.Type(), params)
		} else if fft, _ := checkPtr(f.Type()); fft.Kind() == reflect.Struct {
			v, err = Struct(fname, f.Type(), params)
		}

		if err != nil {
			goto rv
		}

		if v.IsValid() {
			f.Set(v)
		}
	}

rv:
	if isPtr {
		return s.Elem().Addr(), err
	}
	return s.Elem(), err
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//______________________________________________________________________________

func handleTypes(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
	var err error
	var isPtr bool
	typ, isPtr = checkPtr(typ)
	elem := reflect.New(typ).Elem()
	if _, found := params[key]; !found {
		goto rv
	}

	err = parse(params.Get(key), elem)
	if err != nil {
		log.Errorf("Parameter parse error: %s [type: %s, name: %s, value: %s]", err, typ, key, params.Get(key))
		goto rv
	}

rv:
	if isPtr {
		return elem.Addr(), err
	}
	return elem, err
}

func parse(value string, elem reflect.Value) error {
	switch elem.Kind() {
	case reflect.String:
		return parseString(value, elem)
	case reflect.Bool:
		return parseBool(value, elem)
	case reflect.Float32, reflect.Float64:
		return parseFloat(value, elem)
	case reflect.Int, reflect.Int64, reflect.Int8, reflect.Int16, reflect.Int32:
		return parseInt(value, elem)
	case reflect.Uint, reflect.Uint64, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return parseUint(value, elem)
	}

	if elem.Type() == timeType {
		return parseTime(value, elem)
	}

	return nil
}

func handleSlice(key string, typ reflect.Type, params url.Values) (reflect.Value, error) {
	values := params[key]

	// check if it's numbered slice, then create slice values from it
	if len(values) == 0 {
		for k, v := range params {
			if strings.HasPrefix(k, key+"[") {
				values = append(values, v...)
			}
		}
	}

	size := len(values)
	slice := reflect.MakeSlice(typ, size, size)
	if err := parseSlice(values, slice); err != nil {
		log.Errorf("Parameter parse error: %s [type: %s, name: %s, value: %s]", err, typ, key, values)
		return slice, err
	}

	return slice, nil
}

func parseInt(value string, elem reflect.Value) error {
	if value == "" {
		value = "0"
	}

	v, err := strconv.ParseInt(value, 10, getBitSize(elem))
	if err == nil {
		elem.SetInt(v)
	}

	return err
}

func parseUint(value string, elem reflect.Value) error {
	if value == "" {
		value = "0"
	}

	v, err := strconv.ParseUint(value, 10, getBitSize(elem))
	if err == nil {
		elem.SetUint(v)
	}

	return err
}

func parseFloat(value string, elem reflect.Value) error {
	if value == "" {
		value = "0.0"
	}

	v, err := strconv.ParseFloat(value, getBitSize(elem))
	if err == nil {
		elem.SetFloat(v)
	}

	return err
}

func parseBool(value string, elem reflect.Value) error {
	if value == "" {
		value = "0"
	} else if value == "on" || value == "On" {
		value = "1"
	}

	v, err := strconv.ParseBool(value)
	if err == nil {
		elem.SetBool(v)
	}

	return err
}

func parseString(value string, elem reflect.Value) error {
	// Sanitize it; to prevent XSS attacks
	value = template.HTMLEscapeString(value)
	elem.SetString(value)
	return nil
}

func parseSlice(values []string, elem reflect.Value) (err error) {
	for idx := 0; idx < len(values); idx++ {
		el := elem.Index(idx)
		if el.Kind() == reflect.Ptr {
			el.Set(reflect.New(el.Type().Elem()))
			err = parse(values[idx], el.Elem())
		} else {
			err = parse(values[idx], el)
		}
		if err != nil {
			return
		}
	}
	return
}

func parseTime(value string, elem reflect.Value) error {
	if len(strings.TrimSpace(value)) == 0 {
		return nil
	}
	for _, format := range TimeFormats {
		if t, err := time.Parse(format, value); err == nil {
			elem.Set(reflect.ValueOf(t))
			return nil
		}
	}
	return errors.New("valpar: unable to parse time as per 'format.time'")
}

func getBitSize(elem reflect.Value) int {
	switch elem.Kind() {
	case reflect.Int64, reflect.Uint64, reflect.Float64:
		return 64
	case reflect.Int8, reflect.Uint8:
		return 8
	case reflect.Int16, reflect.Uint16:
		return 16
	case reflect.Float32, reflect.Int32, reflect.Uint32:
		return 32
	default:
		return 0 // reflect.Int, reflect.Uint
	}
}

func checkPtr(typ reflect.Type) (reflect.Type, bool) {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem(), true
	}
	return typ, false
}
