// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/valpar source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"errors"

	"gopkg.in/go-playground/validator.v9"
)

// Integrating a library `https://github.com/go-playground/validator` (Version 9)
// as a default validtor.
//
// Currently points at `gopkg.in/go-playground/validator.v9`
var aahValidator *validator.Validate

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Package Methods
//___________________________________

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

// ValidateValue method is to validate individual value on demand.
//
// Returns -
//
//  - true: validation passed
//
//  - false: validation failed
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
func ValidateValue(v interface{}, rules string) bool {
	return Validator().Var(v, rules) == nil
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported Methods
//___________________________________

func checkAndReturn(err error) (validator.ValidationErrors, error) {
	if err != nil {
		if ive, ok := err.(*validator.InvalidValidationError); ok {
			return nil, errors.New(ive.Error())
		}

		return err.(validator.ValidationErrors), nil
	}
	return nil, nil
}
