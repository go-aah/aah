// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package anticsrf

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/test.v0/assert"
)

func TestCSRFSecret(t *testing.T) {
	cfgStr := `
	security {
		anti_csrf {
			#sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    #enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
		}
	}
	`

	cfg, err := config.ParseString(cfgStr)
	assert.Nil(t, err)

	anitCSRF, err := New(cfg)
	assert.Nil(t, err)

	newsecert := anitCSRF.GenerateSecret()
	assert.NotNil(t, newsecert)

	secretstr := anitCSRF.SaltChiperSecret(newsecert)
	assert.NotEqual(t, "", secretstr)

	// Request and Validate
	form := url.Values{}
	form.Set("anti_csrf_token", "XJ3J3I86pzNfHuHW3StkzhtSh_v88le6uXFrsOVxA-eYU8x82XearRHz0wN0SAjJKtBTtC3U_QOk1dpoDd87VA==")

	req, _ := http.NewRequest("POST", "http://localhost:8080/login", strings.NewReader(form.Encode()))
	req.Header.Set(ahttp.HeaderContentType, ahttp.ContentTypeForm.String())
	req.Header.Set("Cookie", "aah_anti_csrf=MTUwNTYwMjUyMHx4TTRGb0ZaTlBaNU83VExWcVdOc0J6R0MxRV9SSnFxNUhhU3gyT2l1T0xNPXw=")
	req.Header.Set(ahttp.HeaderReferer, "http://localhost:8080/login.html")
	_ = req.ParseForm()

	areq := ahttp.AcquireRequest(req)
	areq.Params.Form = req.Form

	assert.False(t, IsSafeHTTPMethod(areq.Method))

	b, _ := url.Parse(req.Header.Get(ahttp.HeaderReferer))
	assert.True(t, IsSameOrigin(req.URL, b))

	secret := anitCSRF.CipherSecret(areq)
	requestSecret := anitCSRF.RequestCipherSecret(areq)
	passed := anitCSRF.IsAuthentic(secret, requestSecret)
	assert.True(t, passed)

	// Write Anti-CSRF cookie
	w := httptest.NewRecorder()
	err = anitCSRF.SetCookie(w, newsecert)
	assert.Nil(t, err)
}
