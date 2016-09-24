// Copyright (c) Jeevanandam M (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

type (
	// pathParam is single URL Path parameter (not a query string values)
	pathParam struct {
		Key   string
		Value string
	}

	// pathParams is a Param-slice, as returned by the route tree.
	pathParams []pathParam
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Path Param methods
//___________________________________

// Get returns the value of the first Path Param which key matches the
// given name. Otherwise an empty string is returned.
func (pp pathParams) Get(name string) string {
	for i := range pp {
		if pp[i].Key == name {
			return pp[i].Value
		}
	}
	return ""
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________
