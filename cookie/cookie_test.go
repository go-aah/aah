// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package cookie

import (
	"testing"

	"aahframework.org/essentials.v0-unstable"
	"aahframework.org/test.v0/assert"
)

func TestCookieNew(t *testing.T) {
	opts := &Options{
		Name:     "aah_cookie",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
	}

	opts.MaxAge = 3600
	cookie := NewWithOptions("This is my cookie for maxage 3600", opts)
	assert.Equal(t, 3600, cookie.MaxAge)

	opts.MaxAge = -1
	cookie = NewWithOptions("This is my cookie for maxage -1", opts)
	assert.Equal(t, -1, cookie.MaxAge)
}

func TestCookieManager(t *testing.T) {
	opts := &Options{
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

	_, err = cm.Decode("MTQ5MTM2OTI4NXxpV1l2SHZrc0tZaXprdlA5Ql9ZS3RWOC1yOFVoWElack1VTGJIM01aV2dGdmJvamJOR2Rmc05KQW1SeHNTS2FoNEJLY2NFN2MyenVCbGllaU1NRFV88hn8MIb0L5HFU6GAkvwYjQ1rvmaL3lG3am2ZageHxQ0=")
	assert.Equal(t, ErrSignVerificationIsFailed, err)

	_, err = cm.Decode("Base64 decode error")
	assert.Equal(t, ess.ErrBase64Decode, err)
}
