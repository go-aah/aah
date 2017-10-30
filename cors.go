// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"net/http"
	"strings"
	"time"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const allowAll = "*"

// CORS struct is to hold Cross-Origin Resource Sharing (CORS) configuration
// values or the route.
type CORS struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           int
}

// AddOrigins method adds the given origin into allow origin list.
func (c *CORS) AddOrigins(origins []string) *CORS {
	for _, o := range origins {
		if o == allowAll {
			c.AllowOrigins = []string{allowAll}
			break
		}
		if !c.isExists(o, c.AllowOrigins) {
			c.AllowOrigins = append(c.AllowOrigins, o)
		}
	}
	return c
}

// AddAllowHeaders method adds the given HTTP header into allow headers list.
func (c *CORS) AddAllowHeaders(hdrs []string) *CORS {
	c.AllowHeaders = c.addHeaders(c.AllowHeaders, hdrs)
	return c
}

// AddAllowMethods method adds the given HTTP verb into allow methods list.
func (c *CORS) AddAllowMethods(methods []string) *CORS {
	for _, m := range methods {
		if m == allowAll {
			c.AllowMethods = []string{allowAll}
			break
		}

		if !ess.IsStrEmpty(m) && !c.isExists(m, c.AllowMethods) {
			c.AllowMethods = append(c.AllowMethods, strings.ToUpper(strings.TrimSpace(m)))
		}
	}
	return c
}

// AddExposeHeaders method adds the given HTTP header into expose headers list.
func (c *CORS) AddExposeHeaders(hdrs []string) *CORS {
	c.ExposeHeaders = c.addHeaders(c.ExposeHeaders, hdrs)
	return c
}

// SetMaxAge method parses the given duration string into seconds and adds to CORS.
// `time.ParseDuration` method time units are supported.
func (c *CORS) SetMaxAge(age string) *CORS {
	if dur, err := time.ParseDuration(age); err == nil {
		c.MaxAge = int(dur.Seconds())
	} else {
		log.Errorf("Unable to parse CORS 'max_age' value '%v'", age)
	}
	return c
}

// SetAllowCredentials method sets the given boolean into allow credentials.
func (c *CORS) SetAllowCredentials(b bool) *CORS {
	c.AllowCredentials = b
	return c
}

// Clone method creates copy of the current CORS config values.
func (c *CORS) Clone() *CORS {
	a := *c
	return &a
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported CORS methods
//___________________________________

func (c *CORS) addHeaders(dst []string, src []string) []string {
	for _, h := range src {
		if h == allowAll {
			return []string{allowAll}
		}
		t := http.CanonicalHeaderKey(strings.TrimSpace(h))
		if !ess.IsStrEmpty(t) && !c.isExists(t, dst) {
			dst = append(dst, t)
		}
	}
	return dst
}

func (c *CORS) isExists(v string, list []string) bool {
	for _, s := range list {
		if v == s {
			return true
		}
	}
	return false
}

func processCORSSection(cfg *config.Config, c *CORS) *CORS {
	var cors CORS
	if c != nil {
		cors = *c
	}

	// Access-Control-Allow-Origin
	if origins, found := cfg.StringList("allow_origins"); found {
		cors.AddOrigins(origins)
	} else {
		cors.AddOrigins([]string{"*"})
	}

	// Access-Control-Allow-Headers
	if hdrs, found := cfg.StringList("allow_headers"); found {
		cors.AddAllowHeaders(hdrs)
		cors.AddAllowHeaders([]string{ahttp.HeaderOrigin})
	} else {
		cors.AddAllowHeaders([]string{"*"})
	}

	// Access-Control-Allow-Methods
	if methods, found := cfg.StringList("allow_methods"); found {
		cors.AddAllowMethods(methods)
	} else {
		cors.AddAllowMethods([]string{"GET", "HEAD", "POST"})
	}

	// Access-Control-Allow-Credentials
	cors.SetAllowCredentials(cfg.BoolDefault("allow_credentials", false))

	// Access-Control-Expose-Headers
	if hdrs, found := cfg.StringList("expose_headers"); found {
		cors.AddExposeHeaders(hdrs)
	}

	// Access-Control-Max-Age
	cors.SetMaxAge(cfg.StringDefault("max_age", "10m"))

	return &cors
}
