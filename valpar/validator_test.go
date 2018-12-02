// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package valpar

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatorValidate(t *testing.T) {
	type testAddress struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
		Planet string `validate:"required"`
		Phone  string `validate:"required"`
	}

	type testValidateStruct1 struct {
		FirstName      string         `validate:"required"`
		LastName       string         `validate:"required"`
		Age            uint8          `validate:"gte=0,lte=130"`
		Email          string         `validate:"required,email"`
		FavouriteColor string         `validate:"iscolor"`                // alias for 'hexcolor|rgb|rgba|hsl|hsla'
		Addresses      []*testAddress `validate:"required,dive,required"` // a person can have a home and cottage...
	}

	address := &testAddress{
		Street: "Eavesdown Docks",
		Planet: "Persphone",
		Phone:  "none",
	}

	testUser := &testValidateStruct1{
		FirstName:      "Badger",
		LastName:       "Smith",
		Age:            135,
		Email:          "Badger.Smith@gmail.com",
		FavouriteColor: "#000-",
		Addresses:      []*testAddress{address},
	}

	result, err := Validate(testUser)
	assert.NotNil(t, result)
	assert.Nil(t, err)

	type testDummy1 struct {
		Street string
		City   string
		Planet string
		Phone  string
	}

	result, err = Validate(testDummy1{})
	assert.Nil(t, result)
	assert.Nil(t, err)

	result, err = Validate(nil)
	assert.Nil(t, result)
	assert.NotNil(t, err)
	assert.Equal(t, "validator: (nil)", err.Error())

	type testStruct struct {
		FirstName      string `bind:"first_name" validate:"required"`
		LastName       string `bind:"last_name" validate:"required"`
		Age            uint8  `bind:"age" validate:"gte=0,lte=130"`
		Email          string `bind:"email" validate:"required,email"`
		FavouriteColor string `bind:"favourite_color" validate:"iscolor"` // alias for 'hexcolor|rgb|rgba|hsl|hsla'
	}

	testUser1 := &testStruct{
		FirstName:      "Badger",
		LastName:       "Smith",
		Age:            135,
		Email:          "Badger.Smith@gmail.com",
		FavouriteColor: "#000-",
	}

	rv := reflect.ValueOf(testUser1)
	result, err = Validate(rv.Interface())
	assert.NotNil(t, result)
	assert.Nil(t, err)
}

func TestValidatorValidateValue(t *testing.T) {
	// Validation failed
	i := 15
	result := ValidateValue(i, "gt=1,lt=10")
	assert.False(t, result)

	emailAddress := "sample@sample"
	result = ValidateValue(emailAddress, "required,email")
	assert.False(t, result)

	numbers := []int{23, 67, 87, 23, 90}
	result = ValidateValue(numbers, "unique")
	assert.False(t, result)

	// validation pass
	i = 9
	result = ValidateValue(i, "gt=1,lt=10")
	assert.True(t, result)

	emailAddress = "sample@sample.com"
	result = ValidateValue(emailAddress, "required,email")
	assert.True(t, result)

	numbers = []int{23, 67, 87, 56, 90}
	result = ValidateValue(numbers, "unique")
	assert.True(t, result)
}

func TestValidatorValidateValues(t *testing.T) {
	values := map[string]string{
		"id":    "5de80bf1-b2c7-4c6e-e47758b7d817",
		"state": "green",
	}

	constraints := map[string]string{
		"id":    "uuid",
		"state": "oneof=3 7 8",
	}

	verrs := map[string]*Error{
		"id":    {Field: "id", Value: "5de80bf1-b2c7-4c6e-e47758b7d817", Constraint: "uuid"},
		"state": {Field: "state", Value: "green", Constraint: "oneof=3 7 8"},
	}

	errs := ValidateValues(values, constraints)
	t.Log(errs.String())
	for _, e := range errs {
		if ev, found := verrs[e.Field]; found {
			assert.Equal(t, ev, e)
		}
	}
}
