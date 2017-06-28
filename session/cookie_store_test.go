// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestSessionGetCookie(t *testing.T) {
	cookieValue := `aah_session=MTQ5MTI4ODQ3NHxHSThCS19qQ2FsbWJ2ZFc3aUNPSUM4RllPRVhTd1had19jS2w0MjE5WU1qLXRLempVeWNhLUFaejhvMEVyY1JmenBLSjRMYXNvd291elN5T2wtMy12dkhRWFlFRThDQmN2VTBnUWZ6UExLaW9zUFFZbnB1YV9VOXJORXNnLWtCT0pOQk5HYzhmVndpR3ZVNUZyRnh4Qy05cHdJOHRNYVJ4YXRGNEtObU94WG1iVnVZM1pJSkdERHpMbzN1VUpxVzgycnZUWWtlbnZUTWdxRDRCTEJEaEhsNHNnZmR3RFJrV1AyUkdfckNFa1lKb2d3VWR3Y0FzS1JtUllPTi0ydHQ3T2JDaUcxQ1JEQUVLbzNUNlRzM1VlUHVTYmtwWUItbFp5czRtd3FGb1VmcHFETkthR2dMWkpHRmM1a1NfZWxXLUljZUdMblJCYTZuTE12NkRvV0ZrQnVYMFFsdUM3clpFdzdUYUFIcFhSaUQ0bHZRS19ZRzExbzlLUTdCVTZnT2xNTmZIal9Oc2VOdWJtd3M3bnlibmlpLTJDRnRkQ1hyU2hYV0pienlTREl1QnRoZHNaQ3lvaGYzbWFCajA0Zi1XcFBwOXF3PT181BI_L4loH_Kcug8MEVnsFj4Ha25umy-8fI0atPVo04k=`

	m := createTestManager(t, `
	security {
	  session {
	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	req := &http.Request{Header: http.Header{}}
	session := m.GetSession(req)
	assert.Nil(t, session)

	req.Header.Add(ahttp.HeaderCookie, cookieValue)
	session = m.GetSession(req)
	assertSessionValue(t, session)
}

func TestSessionCookieStoreSave(t *testing.T) {
	testSessionStoreSave(t, `
	security {
	  session {
	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)
}

func testSessionStoreSave(t *testing.T, cfgStr string) {
	defer ess.DeleteFiles(filepath.Join(getTestdataPath(), "session"))

	m := createTestManager(t, cfgStr)

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	session := m.NewSession()
	session.Set("my-key-1", "my key value 1")
	session.Set("my-key-2", 65454523452)
	session.Set("my-key-3", map[interface{}]interface{}{"test1": "test1value", "test2": 6546546})
	session.Set("my-key-4", float32(364.46))
	session.Set("my-key-5", true)
	session.Set("my-key-6", float64(364534.4637))
	session.Set("my-key-7", int64(65454523452))
	session.SetFlash("my-1", "my 1 flash value")
	session.SetFlash("my-2", 364534.4637)

	w := httptest.NewRecorder()

	err := m.SaveSession(w, session)
	assert.Nil(t, err)

	result := w.Result()
	assert.NotNil(t, result)
	assert.False(t, result.Header.Get(ahttp.HeaderSetCookie) == "")

	// Reading value
	header := http.Header{}
	header.Add(ahttp.HeaderCookie, result.Header.Get(ahttp.HeaderSetCookie))
	req := &http.Request{
		Header: header,
	}

	resultSession := m.GetSession(req)
	assertSessionValue(t, resultSession)
	assert.Equal(t, "my 1 flash value", resultSession.GetFlash("my-1"))
	assert.Equal(t, 364534.4637, resultSession.GetFlash("my-2"))
	assert.Nil(t, resultSession.GetFlash("my-1"))
	assert.Nil(t, resultSession.GetFlash("my-2"))
	assert.Equal(t, float32(364.46), resultSession.GetFloat32("my-key-4"))
	assert.Equal(t, float64(364534.4637), resultSession.GetFloat64("my-key-6"))
	assert.Equal(t, int64(65454523452), resultSession.GetInt64("my-key-7"))
	assert.Equal(t, true, resultSession.GetBool("my-key-5"))
	t.Log(resultSession.String())
}

func TestSessionNewCookie(t *testing.T) {
	m := createTestManager(t, `
	security {
	  session {
	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	opts := *m.Options
	opts.MaxAge = 3600
	cookie := newCookie("This is my cookie for maxage", &opts)
	assert.Equal(t, 3600, cookie.MaxAge)
}
