// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package security

import (
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/security.v0/scheme"
	"aahframework.org/test.v0/assert"
)

func TestSecurityInit(t *testing.T) {
	cfg, err := config.LoadFile(filepath.Join(getTestdataPath(), "security.conf"))
	assert.Nil(t, err)

	sec := New()
	err = sec.Init(cfg)
	assert.Nil(t, err)
	assert.True(t, len(sec.AuthSchemes()) != 0)

	// Add auth scheme
	err = sec.AddAuthScheme("myauth", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "security: auth scheme is nil", err.Error())

	err = sec.AddAuthScheme("form_auth", &scheme.FormAuth{})
	assert.NotNil(t, err)
	assert.Equal(t, "security: auth scheme name 'form_auth' is already added", err.Error())

	// Get auth scheme
	authScheme := sec.AuthScheme("form_auth")
	assert.NotNil(t, authScheme)
	assert.Equal(t, "form", authScheme.Scheme())

	authScheme = sec.AuthScheme("no_auth")
	assert.Nil(t, authScheme)

	// Validate Secure headers
	assert.Equal(t, "SAMEORIGIN", sec.SecureHeaders.Common["X-Frame-Options"])
	assert.Equal(t, "nosniff", sec.SecureHeaders.Common["X-Content-Type-Options"])
	assert.Equal(t, "no-referrer-when-downgrade", sec.SecureHeaders.Common["Referrer-Policy"])
	assert.Equal(t, "master-only", sec.SecureHeaders.Common["X-Permitted-Cross-Domain-Policies"])
	assert.Equal(t, "max-age=31536000; includeSubDomains; preload", sec.SecureHeaders.STS)
	assert.Equal(t, `pin-sha256="X3pGTSOuJeEVw989IJ/cEtXUEmy52zs1TZQrU06KUKg="; pin-sha256="MHJYVThihUrJcxW6wcqyOISTXIsInsdj3xK8QrZbHec="; pin-sha256="GGekerhihUrJcxW6wcqyOISTXIsInsdj3xK8QrZbHec="; max-age=2592000; includeSubdomains; report-uri=http://report.localhost`, sec.SecureHeaders.PKP)
	assert.Equal(t, "1; mode=block", sec.SecureHeaders.XSSFilter)
	assert.Equal(t, "default-src 'none'; report-uri http://report.localhost", sec.SecureHeaders.CSP)
	assert.True(t, sec.SecureHeaders.CSPReportOnly)
	assert.True(t, sec.SecureHeaders.PKPReportOnly)
}

func TestSecurityInitError(t *testing.T) {
	cfg, err := config.ParseString(`
		security {
		  auth_schemes {
		    # HTTP Form Auth Scheme
		    form_auth {
					# for error
		      #scheme = "form"

		      # Authenticator is used to validate the subject (aka User)
		      authenticator = "security/Authentication"

		      # Authorizer is used to get Subject authorization information,
		      # such as Roles and Permissions
		      authorizer = "security/Authorization"
		    }
		  }
		}
	`)
	assert.Nil(t, err)

	sec1 := New()
	err = sec1.Init(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "security: 'security.auth_schemes.form_auth.scheme' is required", err.Error())

	cfg, err = config.ParseString(`
		security {
		  auth_schemes {
		    # HTTP Form Auth Scheme
		    form_auth {
		      scheme = "unknown"

		      # Authenticator is used to validate the subject (aka User)
		      authenticator = "security/Authentication"

		      # Authorizer is used to get Subject authorization information,
		      # such as Roles and Permissions
		      authorizer = "security/Authorization"
		    }
		  }
		}
	`)
	assert.Nil(t, err)

	sec2 := New()
	err = sec2.Init(cfg)
	assert.NotNil(t, err)
	assert.Equal(t, "security: auth scheme 'unknown' not available", err.Error())

	result := parseToSecondsString("2", 60)
	assert.Equal(t, "60", result)
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
