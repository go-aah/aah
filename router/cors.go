// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aahframe.work/aah/ahttp"
	"aahframe.work/aah/config"
	"aahframe.work/aah/essentials"
	"aahframe.work/aah/log"
)

const allowAll = "*"

// CORS errors
var (
	ErrCORSOriginIsInvalid       = errors.New("cors: invalid origin")
	ErrCORSMethodNotAllowed      = errors.New("cors: method not allowed")
	ErrCORSHeaderNotAllowed      = errors.New("cors: header not allowed")
	ErrCORSContentTypeNotAllowed = errors.New("cors: content-type not allowed")

	// Excluded simple allowed headers and adding sensiable allowed headers.
	// Refer to: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Access-Control-Allow-Headers
	defaultAllowHeaders = []string{ahttp.HeaderOrigin, ahttp.HeaderAccept,
		ahttp.HeaderAcceptLanguage, ahttp.HeaderAuthorization}

	// Sensiable default allowed methods
	defaultAllowMethods = []string{ahttp.MethodGet, ahttp.MethodHead, ahttp.MethodPost}
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// CORS
//______________________________________________________________________________

// CORS struct holds Cross-Origin Resource Sharing (CORS) configuration
// values and verification methods for the route.
//
// Spec: https://www.w3.org/TR/cors/
// Friendly Read: https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
type CORS struct {
	AllowCredentials bool
	allowAllOrigins  bool
	allowAllMethods  bool
	allowAllHeaders  bool

	MaxAge        string
	maxAgeStr     string
	AllowOrigins  []string
	AllowMethods  []string
	AllowHeaders  []string
	ExposeHeaders []string
}

// AddOrigins method adds the given origin into allow origin list.
func (c *CORS) AddOrigins(origins []string) *CORS {
	for _, o := range origins {
		if o == allowAll {
			c.allowAllOrigins = true
			c.AllowOrigins = []string{allowAll}
			break
		}
		o = strings.ToLower(o)
		if !ess.IsSliceContainsString(c.AllowOrigins, o) {
			c.AllowOrigins = append(c.AllowOrigins, o)
		}
	}
	return c
}

// AddAllowHeaders method adds the given HTTP header into allow headers list.
func (c *CORS) AddAllowHeaders(hdrs []string) *CORS {
	c.AllowHeaders = c.addHeaders(c.AllowHeaders, hdrs)
	if ess.IsSliceContainsString(c.AllowHeaders, allowAll) {
		c.allowAllHeaders = true
		c.AllowHeaders = []string{allowAll}
	}
	return c
}

// AddAllowMethods method adds the given HTTP verb into allow methods list.
func (c *CORS) AddAllowMethods(methods []string) *CORS {
	for _, m := range methods {
		if m == allowAll {
			c.allowAllMethods = true
			c.AllowMethods = []string{allowAll}
			break
		}
		m = strings.ToUpper(strings.TrimSpace(m))
		if !ess.IsStrEmpty(m) && !ess.IsSliceContainsString(c.AllowMethods, m) {
			c.AllowMethods = append(c.AllowMethods, m)
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
		c.MaxAge = strconv.Itoa(int(dur.Seconds()))
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

// IsOriginAllowed method check given origin is allowed or not.
func (c *CORS) IsOriginAllowed(origin string) bool {
	if len(origin) == 0 {
		return false
	}

	if c.allowAllOrigins {
		return true
	}

	return ess.IsSliceContainsString(c.AllowOrigins, strings.ToLower(origin))
}

// IsMethodAllowed method returns true if preflight method is allowed otherwise
// false.
func (c *CORS) IsMethodAllowed(method string) bool {
	if c.allowAllMethods {
		return true
	}
	return ess.IsSliceContainsString(c.AllowMethods, method)
}

// IsHeadersAllowed method returns true if preflight headers are allowed otherwise
// false.
func (c *CORS) IsHeadersAllowed(hdrs string) bool {
	if c.allowAllHeaders || len(hdrs) == 0 {
		return true
	}

	for _, h := range strings.Split(hdrs, ",") {
		h = http.CanonicalHeaderKey(strings.TrimSpace(h))
		allowed := false
		if ess.IsSliceContainsString(c.AllowHeaders, h) {
			allowed = true
		}

		if !allowed {
			return false
		}
	}

	return true
}

// String method returns string representation of CORS configuration values.
func (c CORS) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("cors(allow-origins:")
	buf.WriteString(strings.Join(c.AllowOrigins, ","))
	buf.WriteString(" allow-headers:")
	buf.WriteString(strings.Join(c.AllowHeaders, ","))
	buf.WriteString(" allow-methods:")
	buf.WriteString(strings.Join(c.AllowMethods, ","))
	buf.WriteString(" expose-headers:")
	buf.WriteString(strings.Join(c.ExposeHeaders, ","))
	buf.WriteString(fmt.Sprintf(" allow-credentials:%v", c.AllowCredentials))
	buf.WriteString(fmt.Sprintf(" max-age:%s", c.maxAgeStr))
	buf.WriteByte(')')
	return buf.String()
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported CORS methods
//______________________________________________________________________________

func (c *CORS) addHeaders(dst []string, src []string) []string {
	for _, h := range src {
		if h == allowAll {
			return []string{allowAll}
		}
		h = http.CanonicalHeaderKey(strings.TrimSpace(h))
		if !ess.IsStrEmpty(h) && !ess.IsSliceContainsString(dst, h) {
			dst = append(dst, h)
		}
	}
	return dst
}

func processBaseCORSSection(cfg *config.Config) *CORS {
	cors := &CORS{}

	// Access-Control-Allow-Origin
	if origins, found := cfg.StringList("allow_origins"); found {
		cors.AddOrigins(origins)
	} else {
		cors.AddOrigins([]string{allowAll})
	}

	// Access-Control-Allow-Headers
	if hdrs, found := cfg.StringList("allow_headers"); found {
		cors.AddAllowHeaders(hdrs)
	} else {
		cors.AddAllowHeaders(defaultAllowHeaders)
	}

	// Access-Control-Allow-Methods
	if methods, found := cfg.StringList("allow_methods"); found {
		cors.AddAllowMethods(methods)
	} else {
		cors.AddAllowMethods(defaultAllowMethods)
	}
	cors.AddAllowMethods([]string{ahttp.MethodOptions})

	// Access-Control-Allow-Credentials
	cors.SetAllowCredentials(cfg.BoolDefault("allow_credentials", false))

	// Access-Control-Expose-Headers
	if hdrs, found := cfg.StringList("expose_headers"); found {
		cors.AddExposeHeaders(hdrs)
	}

	// Access-Control-Max-Age
	cors.maxAgeStr = cfg.StringDefault("max_age", "24h")
	cors.SetMaxAge(cors.maxAgeStr)

	return cors
}

func processCORSSection(cfg *config.Config, parent *CORS) *CORS {
	cors := &CORS{}

	// Access-Control-Allow-Origin
	if origins, found := cfg.StringList("allow_origins"); found {
		cors.AddOrigins(origins)
	} else {
		cors.AddOrigins(parent.AllowOrigins)
	}

	// Access-Control-Allow-Headers
	if hdrs, found := cfg.StringList("allow_headers"); found {
		cors.AddAllowHeaders(hdrs)
	} else {
		cors.AddAllowHeaders(parent.AllowHeaders)
	}

	// Access-Control-Allow-Methods
	if methods, found := cfg.StringList("allow_methods"); found {
		cors.AddAllowMethods(methods)
	} else {
		cors.AddAllowMethods(parent.AllowMethods)
	}

	// Access-Control-Allow-Credentials
	cors.SetAllowCredentials(cfg.BoolDefault("allow_credentials", parent.AllowCredentials))

	// Access-Control-Expose-Headers
	if hdrs, found := cfg.StringList("expose_headers"); found {
		cors.AddExposeHeaders(hdrs)
	} else {
		cors.AddExposeHeaders(parent.ExposeHeaders)
	}

	// Access-Control-Max-Age
	cors.maxAgeStr = cfg.StringDefault("max_age", parent.maxAgeStr)
	cors.SetMaxAge(cors.maxAgeStr)

	return cors
}
