// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/go-playground/validator.v9"
)

// Validator ...
type Validator interface {
	// Validate method is to validate all struct fields with tagged constraints
	// for fields.
	//
	// Returns:
	//
	//  - For validation errors: returns `valpar.Errors` and nil
	//
	//  - For invalid input: returns `valpar.InvalidValue` (invalid input such as nil, non-struct, etc.)
	//
	//  - For no validation errors: nil
	Validate(s interface{}) error

	// ValidateExcept method is to validate all struct fields with tagged constraints
	// except the ones passed in. Fields may be provided in a namespace path
	// relative to the struct provided i.e. NestedStruct.Field or NestedArrayField[0].Struct.Name
	//
	// Returns:
	//
	//  - For validation errors: returns `valpar.Errors` and nil
	//
	//  - For invalid input: returns `valpar.InvalidValue` (invalid input such as nil, non-struct, etc.)
	//
	//  - For no validation errors: nil
	ValidateExcept(s interface{}, fields ...string) error

	// ValidatePartial method is to validate fields passed in againsts struct.
	//
	// Returns:
	//
	//  - For validation errors: returns `valpar.Errors` and nil
	//
	//  - For invalid input: returns `valpar.InvalidValue` (invalid input such as nil, non-struct, etc.)
	//
	//  - For no validation errors: nil
	ValidatePartial(s interface{}, fields ...string) error

	// ValidateValue method is to validate individual value. Returns true if
	// constraint passed otherwise false.
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
	ValidateValue(val interface{}, constraint string) bool

	// RegisterConstriant adds user defined constraint function into validator registry.
	//
	// Notes: If the given constraint name exists already, it overwrites the
	// previous constraint function. Also its not goroutine safe.
	RegisterConstriant(constraint string, fn ConstraintFunc)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Validate type and its methods
//______________________________________________________________________________

var _ Validator = (*Validate)(nil)

// Validate ...
type Validate struct {
	v *validator.Validate
}

// Validate ...
func (v *Validate) Validate(s interface{}) error {
	err := v.v.Struct(s)
	if err != nil {
		if ive, ok := err.(*validator.InvalidValidationError); ok {
			return &InvaildValue{Type: ive.Type}
		}
		var errs Errors
		for _, fe := range err.(validator.ValidationErrors) {
			fmt.Println(fe.Namespace())
			fmt.Println(fe.Field())
			fmt.Println(fe.StructNamespace()) // can differ when a custom TagNameFunc is registered or
			fmt.Println(fe.StructField())     // by passing alt name to ReportError like below
			fmt.Println(fe.Tag())
			fmt.Println(fe.ActualTag())
			fmt.Println(fe.Kind())
			fmt.Println(fe.Type())
			fmt.Println(fe.Value())
			fmt.Println(fe.Param())
			fmt.Println()
			errs = append(errs, &fieldError{e: fe})
		}
		return errs
	}
	return nil
}

// ValidateExcept method doc refer to `valpar.Validator`.
func (v *Validate) ValidateExcept(s interface{}, fields ...string) error {
	return nil
}

// ValidatePartial method doc refer to `valpar.Validator`.
func (v *Validate) ValidatePartial(s interface{}, fields ...string) error {
	return nil
}

// ValidateValue method doc refer to `valpar.Validator`.
func (v *Validate) ValidateValue(val interface{}, constraint string) bool {
	return false
}

// RegisterConstriant method doc refer to `valpar.Validator`.
func (v *Validate) RegisterConstriant(constraint string, fn ConstraintFunc) {
	// v.v.RegisterValidation(constraint, fn)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Field type and its methods
//______________________________________________________________________________

// Field ...
type Field interface {
	// Name method returns the field name. Tag name
	// takes precedence over the actual field name.
	Name() string

	// Value method returns current field for validation
	Value() reflect.Value

	// Constriant method returns the validate constraint placed for
	// the field/supplied value.
	//
	// For e.g.: `iscolor`
	Constraint() string

	// Parent method returns the current fields parent struct, if any
	Parent() reflect.Value

	// ExtractType gets the actual underlying type of field value.
	// It will dive into pointers, customTypes and return you the
	// underlying value and it's kind.
	// ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)

	// traverses the parent struct to retrieve a specific field denoted by the provided namespace
	// in the param and returns the field, field kind and whether is was successful in retrieving
	// the field at all.
	//
	// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
	// could not be retrieved because it didn't exist.
	// GetStructFieldOK() (reflect.Value, reflect.Kind, bool)
}

var _ Error = (*fieldError)(nil)

type fieldError struct {
	e validator.FieldError
}

func (fe *fieldError) Name() string            { return fe.e.Field() }
func (fe *fieldError) Value() interface{}      { return fe.e.Value() }
func (fe *fieldError) Constraint() string      { return fe.e.Tag() }
func (fe *fieldError) ConstraintValue() string { return fe.e.Param() }
func (fe *fieldError) Namespace() string       { return fe.e.Namespace() }

func (fe *fieldError) I18nKey() string {
	return ""
}

func (fe *fieldError) I18nMessage() string {
	return ""
}

func (fe *fieldError) Error() string {
	return fmt.Sprintf("key: '%s' error:field validation for '%s' failed on the '%s' constraint",
		fe.Namespace(), fe.Name(), fe.Constraint())
}

// ConstraintFunc ...
type ConstraintFunc func(f Field) bool

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// InvaildValue type and its methods
//______________________________________________________________________________

// InvaildValue ...
type InvaildValue struct {
	Type reflect.Type
}

// Error method returns InvalidValue error message
func (v *InvaildValue) Error() string {
	if v.Type == nil {
		return "validator: (nil)"
	}
	return "validator: (nil " + v.Type.String() + ")"
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Error types, definitions and its methods
//______________________________________________________________________________

// Errors ...
type Errors []Error

func (e Errors) Error() string {
	buff := bytes.NewBufferString("")
	var fe *fieldError
	for i := 0; i < len(e); i++ {
		fe = e[i].(*fieldError)
		buff.WriteString(fe.Error())
		buff.WriteString("\n")
	}
	return strings.TrimSpace(buff.String())
}

// Error ...
type Error interface {
	// Name method returns the name of the field. Tag
	// name takes precedence over the actual field name.
	//
	// For e.g.:
	//
	// 	type Address struct {
	// 		Street string `json:"street" validate:"required"`
	// 	}
	//
	// So return value is `street`
	Name() string

	// Value method returns the actual field value.
	Value() interface{}

	// Constriant method returns the validate constraint placed for
	// the field/supplied value.
	//
	// For e.g.: `iscolor`
	Constraint() string

	// Constriant method returns the constraint value defined for
	// the field/supplied value.
	//
	// For e.g.: `gte=0,lte=130`
	ConstraintValue() string

	// I18nKey method returns the i18n message key.
	//
	// For e.g.:
	//
	// 	type Address struct {
	// 		Street string `json:"street" validate:"required" i18n:"user.address.street"`
	// 	}
	//
	// So return value is `user.address.street`
	I18nKey() string

	// I18nMessage method returns the i18n message.
	I18nMessage() string

	// Namespace method returns the namespace of the field. Tag
	// name takes precedence over the actual field name.
	//
	// For e.g.:
	//
	// 	type Address struct {
	// 		Street string `json:"street" validate:"required"`
	// 	}
	//
	// So return value is `Address.street`
	Namespace() string
}

// // Integrating a library `https://github.com/go-playground/validator` (Version 9)
// // as a validtor.
// //
// // Currently points at `gopkg.in/go-playground/validator.v9`
// var aahValidator *validator.Validate

// //‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// // Package methods
// //______________________________________________________________________________

// // Validator method return the default validator of aah framework.
// //
// // Refer to https://godoc.org/gopkg.in/go-playground/validator.v9 for detailed
// // documentation.
// func Validator() *validator.Validate {
// 	if aahValidator == nil {
// 		aahValidator = validator.New()

// 		// Do customizations here
// 	}
// 	return aahValidator
// }

// // Validate method is to validate struct via underneath validator.
// //
// // Returns:
// //
// //  - For validation errors: returns `validator.ValidationErrors` and nil
// //
// //  - For invalid input: returns nil, error (invalid input such as nil, non-struct, etc.)
// //
// //  - For no validation errors: nil, nil
// func Validate(s interface{}) (validator.ValidationErrors, error) {
// 	return checkAndReturn(Validator().Struct(s))
// }

// // ValidateValue method is to validate individual value. Returns true if
// // validation is passed otherwise false.
// //
// // For example:
// //
// // 	i := 15
// // 	result := valpar.ValidateValue(i, "gt=1,lt=10")
// //
// // 	emailAddress := "sample@sample"
// // 	result := valpar.ValidateValue(emailAddress, "email")
// //
// // 	numbers := []int{23, 67, 87, 23, 90}
// // 	result := valpar.ValidateValue(numbers, "unique")
// func ValidateValue(v interface{}, constraint string) bool {
// 	return Validator().Var(v, constraint) == nil
// }

// // ValidateValues method validates the values with respective constraints.
// // Returns nil if no errors otherwise slice of error.
// func ValidateValues(values map[string]string, constraints map[string]string) Errors {
// 	var errs Errors
// 	for k, v := range values {
// 		if !ValidateValue(v, constraints[k]) {
// 			errs = append(errs, &Error{
// 				Field:      k,
// 				Value:      v,
// 				Constraint: constraints[k],
// 			})
// 		}
// 	}
// 	return errs
// }

// //‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// // Error type and its methods
// //______________________________________________________________________________

// // Errors type represents list errors.
// type Errors []*Error

// // String is Stringer interface.
// func (e Errors) String() string {
// 	if len(e) == 0 {
// 		return ""
// 	}

// 	var errs []string
// 	for _, er := range e {
// 		errs = append(errs, er.String())
// 	}
// 	return strings.Join(errs, ",")
// }

// // Error represents single validation error details.
// type Error struct {
// 	Field      string
// 	Value      string
// 	Key        string // i18n key
// 	Msg        string // i18n message
// 	Constraint string
// }

// // String is Stringer interface.
// func (e Error) String() string {
// 	return fmt.Sprintf("error(field:%s value:%v key:%s msg:%v constraint:%s)",
// 		e.Field, e.Value, e.Key, e.Msg, e.Constraint)
// }

// //‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// // Unexported methods
// //______________________________________________________________________________

// func checkAndReturn(err error) (validator.ValidationErrors, error) {
// 	if err != nil {
// 		if ive, ok := err.(*validator.InvalidValidationError); ok {
// 			return nil, errors.New(ive.Error())
// 		}

// 		return err.(validator.ValidationErrors), nil
// 	}
// 	return nil, nil
// }
