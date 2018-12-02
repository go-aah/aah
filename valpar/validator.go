// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/go-playground/validator.v9"
)

// Integrating a library `https://github.com/go-playground/validator` (Version 9)
// as a validtor.
//
// Currently points at `gopkg.in/go-playground/validator.v9`
var aahValidator *validator.Validate

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package methods
//______________________________________________________________________________

// Validator method return the default validator of aah framework.
//
// Refer to https://godoc.org/gopkg.in/go-playground/validator.v9 for detailed
// documentation.
func Validator() *validator.Validate {
	if aahValidator == nil {
		aahValidator = validator.New()

		// Do customizations here
	}
	return aahValidator
}

// Validate method is to validate struct via underneath validator.
//
// Returns:
//
//  - For validation errors: returns `validator.ValidationErrors` and nil
//
//  - For invalid input: returns nil, error (invalid input such as nil, non-struct, etc.)
//
//  - For no validation errors: nil, nil
func Validate(s interface{}) (validator.ValidationErrors, error) {
	return checkAndReturn(Validator().Struct(s))
}

// ValidateValue method is to validate individual value. Returns true if
// validation is passed otherwise false.
//
// For example:
//
// 	i := 15
// 	result := valpar.ValidateValue(i, "gt=1,lt=10")
//
// 	emailAddress := "sample@sample"
// 	result := valpar.ValidateValue(emailAddress, "email")
//
// 	numbers := []int{23, 67, 87, 23, 90}
// 	result := valpar.ValidateValue(numbers, "unique")
func ValidateValue(v interface{}, constraint string) bool {
	return Validator().Var(v, constraint) == nil
}

// ValidateValues method validates the values with respective constraints.
// Returns nil if no errors otherwise slice of error.
func ValidateValues(values map[string]string, constraints map[string]string) Errors {
	var errs Errors
	for k, v := range values {
		if !ValidateValue(v, constraints[k]) {
			errs = append(errs, &Error{
				Field:      k,
				Value:      v,
				Constraint: constraints[k],
			})
		}
	}
	return errs
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Error type and its methods
//______________________________________________________________________________

// Errors type represents list errors.
type Errors []*Error

// String is Stringer interface.
func (e Errors) String() string {
	if len(e) == 0 {
		return ""
	}

	var errs []string
	for _, er := range e {
		errs = append(errs, er.String())
	}
	return strings.Join(errs, ",")
}

// Error represents single validation error details.
type Error struct {
	Field      string
	Value      string
	Key        string // i18n key
	Msg        string // i18n message
	Constraint string
}

// String is Stringer interface.
func (e Error) String() string {
	return fmt.Sprintf("error(field:%s value:%v key:%s msg:%v constraint:%s)",
		e.Field, e.Value, e.Key, e.Msg, e.Constraint)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//______________________________________________________________________________

func checkAndReturn(err error) (validator.ValidationErrors, error) {
	if err != nil {
		if ive, ok := err.(*validator.InvalidValidationError); ok {
			return nil, errors.New(ive.Error())
		}

		return err.(validator.ValidationErrors), nil
	}
	return nil, nil
}
