// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cookie

import (
	"net/http/httptest"
	"strings"
	"testing"

	"aahframe.work/essentials"
	"github.com/stretchr/testify/assert"
)

func TestCookieNew(t *testing.T) {
	opts := &Options{
		Name:     "aah_cookie",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	}

	opts.MaxAge = 3600
	cookie := NewWithOptions("This is my cookie for maxage 3600", opts)
	assert.Equal(t, 3600, cookie.MaxAge)

	opts.MaxAge = -1
	cookie = NewWithOptions("This is my cookie for maxage -1", opts)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestCookieWithNoKeys(t *testing.T) {
	cm, err := NewManager(&Options{
		Name:   "aah",
		MaxAge: 1800,
	})
	assert.Nil(t, err)
	assert.NotNil(t, cm)

	value := "im the cookie with no keys"

	encodeValue, err := cm.Encode([]byte(value))
	assert.Nil(t, err)
	assert.True(t, len(encodeValue) > 0)

	c1 := cm.New(encodeValue)
	assert.NotNil(t, c1)
	assert.True(t, len(c1.Value) > 0)

	r1, err := cm.Decode(encodeValue)
	assert.Nil(t, err)
	assert.Equal(t, value, string(r1))
}

func TestCookieWithKeys(t *testing.T) {
	opts := &Options{
		Name:   "aah",
		MaxAge: 1800,
	}

	cm, err := NewManager(opts, "eFWLXEewECptbDVXExokRTLONWxrTjfV", "KYqklJsgeclPpZutTeQKNOTWlpksRBwA")
	assert.Nil(t, err)
	assert.NotNil(t, cm)

	value := "This is testing of encode and decode value with sign and encryption"
	result, err := cm.Encode([]byte(value))
	assert.Nil(t, err)
	assert.NotNil(t, result)

	obj, err := cm.Decode(result)
	assert.Nil(t, err)
	assert.Equal(t, value, string(obj))

	w := httptest.NewRecorder()
	cm.Write(w, value)
	hdr := w.Header().Get("Set-Cookie")
	assert.True(t, strings.Contains(hdr, value))

	cookie := cm.New(result)
	assert.NotNil(t, cookie)

	r2, err := cm.Decode(result)
	assert.Nil(t, err)
	assert.Equal(t, value, string(r2))

	_, err = cm.Decode("MTQ5MTM2OTI4NXxpV1l2SHZrc0tZaXprdlA5Ql9ZS3RWOC1yOFVoWElack1VTGJIM01aV2dGdmJvamJOR2Rmc05KQW1SeHNTS2FoNEJLY2NFN2MyenVCbGllaU1NRFV88hn8MIb0L5HFU6GAkvwYjQ1rvmaL3lG3am2ZageHxQ0=")
	assert.Equal(t, ErrSignVerificationIsFailed, err)

	_, err = cm.Decode("Base64 decode error")
	assert.Equal(t, ess.ErrBase64Decode, err)

	bvalue, _ := ess.DecodeBase64([]byte(value))
	_, err = cm.Decode(string(bvalue))
	assert.Equal(t, ErrCookieValueIsInvalid, err)

	var dvalue string
	for i := 0; i < 70; i++ {
		dvalue = dvalue + value
	}
	_, err = cm.Decode(dvalue)
	assert.Equal(t, ErrCookieValueIsTooLarge, err)
}

func TestCookieWithKeysRotation(t *testing.T) {
	opts := &Options{
		Name:     "aah",
		MaxAge:   1 << 31,
		SameSite: "lax",
	}

	cmo, _ := NewManager(opts, "eFWLXEewECptbDVXExokRTLONWxrTold", "KYqklJsgeclPpZutTeQKNOTWlpksRold")
	assert.NotNil(t, cmo)

	value := "im the cookie with new and old keys"
	c1 := cmo.New(value)
	assert.NotNil(t, c1)
	assert.Equal(t, value, c1.Value)

	encodeValue, err := cmo.Encode([]byte(value))
	assert.Nil(t, err)
	assert.True(t, len(encodeValue) > 0)

	r1, err := cmo.Decode(encodeValue)
	assert.Nil(t, err)
	assert.Equal(t, value, string(r1))

	opts.SameSite = "strict"
	cmr, err := NewManager(opts, "eFWLXEewECptbDVXExokRTLONWxrTjfV", "KYqklJsgeclPpZutTeQKNOTWlpksRBwA",
		"eFWLXEewECptbDVXExokRTLONWxrTold", "KYqklJsgeclPpZutTeQKNOTWlpksRold")
	assert.Nil(t, err)

	c2 := cmr.New(value)
	assert.NotNil(t, c1)
	assert.Equal(t, value, c2.Value)

	r2, err := cmr.Decode(encodeValue)
	assert.Nil(t, err)
	assert.Equal(t, value, string(r2))

	_, err = cmr.Decode("MTQ5MTM2OTI4NXxpV1l2SHZrc0tZaXprdlA5Ql9ZS3RWOC1yOFVoWElack1VTGJIM01aV2dGdmJvamJOR2Rmc05KQW1SeHNTS2FoNEJLY2NFN2MyenVCbGllaU1NRFV88hn8MIb0L5HFU6GAkvwYjQ1rvmaL3lG3am2ZageHxQ0=")
	assert.NotNil(t, err)
	assert.Equal(t, ErrSignVerificationIsFailed, err)
}
