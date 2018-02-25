// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package anticsrf

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestAntiCSRFNotEnabled(t *testing.T) {
	cfgStr := `
	security {
	}
	`

	cfg, err := config.ParseString(cfgStr)
	assert.Nil(t, err)

	antiCSRF, err := New(cfg)
	assert.Nil(t, err)

	assert.False(t, antiCSRF.Enabled)
}

func TestAntiCSRFSecret(t *testing.T) {
	cfgStr := `
	security {
		anti_csrf {
			enable = true
			sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
		}
	}
	`

	cfg, err := config.ParseString(cfgStr)
	assert.Nil(t, err)

	antiCSRF, err := New(cfg)
	assert.Nil(t, err)

	newsecret := antiCSRF.GenerateSecret()
	secretstr := antiCSRF.SaltCipherSecret(newsecret)
	decodesecret, _ := ess.DecodeBase64([]byte(secretstr))
	testsecret := antiCSRF.unsaltCipherToken(decodesecret)
	assert.NotNil(t, newsecret)
	assert.NotEqual(t, "", secretstr)
	assert.True(t, bytes.Equal(testsecret, newsecret))

	// Request and Validate
	cookieValue, _ := antiCSRF.cookieMgr.Encode(newsecret)
	form := url.Values{}
	form.Set("anti_csrf_token", secretstr)
	req, _ := http.NewRequest("POST", "http://localhost:8080/login", strings.NewReader(form.Encode()))
	req.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	req.Header.Set(ahttp.HeaderCookie, "aah_anti_csrf="+cookieValue)
	req.Header.Set(ahttp.HeaderReferer, "http://localhost:8080/login.html")
	_ = req.ParseForm()

	areq := ahttp.AcquireRequest(req)
	areq.Params.Form = req.Form

	secret := antiCSRF.CipherSecret(areq)
	requestSecret := antiCSRF.RequestCipherSecret(areq)
	assert.True(t, bytes.Equal(secret, requestSecret))

	result := antiCSRF.IsAuthentic(secret, requestSecret)
	assert.True(t, result)

	// Write Anti-CSRF cookie
	w := httptest.NewRecorder()
	err = antiCSRF.SetCookie(w, newsecret)
	assert.Nil(t, err)

	antiCSRF.ClearCookie(w, areq)

	// Safe method check
	assert.False(t, IsSafeHTTPMethod(areq.Method))

	// same origin check
	b, _ := url.Parse(req.Header.Get(ahttp.HeaderReferer))
	assert.True(t, IsSameOrigin(req.URL, b))
}

func TestAntiCSRFCipherSecret(t *testing.T) {
	cfgStr := `
	security {
		anti_csrf {
			enable = true
			sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
		}
	}
	`

	cfg, err := config.ParseString(cfgStr)
	assert.Nil(t, err)

	antiCSRF, err := New(cfg)
	assert.Nil(t, err)

	req, _ := http.NewRequest("GET", "http://localhost:8080/login.html", nil)
	areq := ahttp.AcquireRequest(req)

	secret := antiCSRF.CipherSecret(areq)
	assert.NotNil(t, secret)

	areq.Unwrap().Header.Set("Cookie", "aah_anti_csrf=This is cookie value")
	secret = antiCSRF.CipherSecret(areq)
	assert.NotNil(t, secret)
}
