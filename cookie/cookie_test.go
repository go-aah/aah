package cookie

import (
	"testing"

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
