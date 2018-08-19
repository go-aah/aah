// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/essentials source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package ess

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLValue(t *testing.T) {
	t.Log("Valid URLs")
	for _, u := range []string{
		"https://aahframework.org",
		"/facebook-auth/callback?code=AQD5-i29_Vn7fNg8VgcV-Uzk_QNO1sx6yO-tJvkRiFaGs1n-wnCxUnvLX0V2q25Tx7JRZAiTds2-DIKrDb8jPEdGquMCedQ_mpMZQxsHmPYeg_cP1Xjy2jHooK-1ZDJZQHXtDSL8FA7r3nVA7WcrCuLZrlrgXq8LnOSAil3oMD-RqPix-nI576GvAPGgiXo6ep_AfS2GaF8A8TOTwl2iwEjB74F23yEukNpz5tmDJVfH02qtrGECuDfaAEPc-4u2wVIIWKCWy3oEEeoEr4zBdzsMtUR1FP3X5yUm0_yAYFP3taAPrpM-5UqtJWmUgaOY-U0&state=72du0BBFIPjW4PsWcI21SccfRX-EkojC4dZ49MYJZac",
		"https://www.facebook.com/dialog/oauth?client_id=182958108394860&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Ffacebook-auth%2Fcallback&response_type=code&scope=public_profile+email&state=72du0BBFIPjW4PsWcI21SccfRX-EkojC4dZ49MYJZac",
		"https://godoc.org/golang.org/x/oauth2",
		"http://godoc.org/",
	} {
		assert.True(t, IsVaildURL(u))
	}

	t.Log("Absolute URL")
	for _, u := range []string{
		"https://aahframework.org",
		"https://www.facebook.com/dialog/oauth?client_id=182958108394860&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Ffacebook-auth%2Fcallback&response_type=code&scope=public_profile+email&state=72du0BBFIPjW4PsWcI21SccfRX-EkojC4dZ49MYJZac",
		"https://godoc.org/golang.org/x/oauth2",
		"http://godoc.org/",
	} {
		assert.True(t, IsAbsURL(u))
	}

	t.Log("Error - Absolute URL")
	for _, u := range []string{
		"://aahframework.org",
		"://",
		"http",
		"https",
	} {
		assert.False(t, IsAbsURL(u))
	}

	t.Log("Relative URL")
	for _, u := range []string{
		"/facebook-auth/callback?code=AQD5-i29_Vn7fNg8VgcV-Uzk_QNO1sx6yO-tJvkRiFaGs1n-wnCxUnvLX0V2q25Tx7JRZAiTds2-DIKrDb8jPEdGquMCedQ_mpMZQxsHmPYeg_cP1Xjy2jHooK-1ZDJZQHXtDSL8FA7r3nVA7WcrCuLZrlrgXq8LnOSAil3oMD-RqPix-nI576GvAPGgiXo6ep_AfS2GaF8A8TOTwl2iwEjB74F23yEukNpz5tmDJVfH02qtrGECuDfaAEPc-4u2wVIIWKCWy3oEEeoEr4zBdzsMtUR1FP3X5yUm0_yAYFP3taAPrpM-5UqtJWmUgaOY-U0&state=72du0BBFIPjW4PsWcI21SccfRX-EkojC4dZ49MYJZac",
		"/dialog/oauth?client_id=182958108394860&redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Ffacebook-auth%2Fcallback&response_type=code&scope=public_profile+email&state=72du0BBFIPjW4PsWcI21SccfRX-EkojC4dZ49MYJZac",
		"/golang.org/x/oauth2",
	} {
		assert.True(t, IsRelativeURL(u))
	}

}
