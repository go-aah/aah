// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"encoding/gob"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/test.v0/assert"
)

func TestSessionFileStoreGet(t *testing.T) {
	defer ess.DeleteFiles(filepath.Join(getTestdataPath(), "session"))

	m := createTestManager(t, `
	security {
	  session {
	    store {
	      type = "file"
	      filepath = "testdata/session"
	    }

	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	// Session ID is SWkGHtLck_sv7kWKDvvN8mwSq3CPfmkoRkz1POMtnx8
	cookieValue := "aah_session=MTQ5MTM0MDUyMHxNamd3MlgxRFFkOUIzSjFJa0RROXhkOW9tS0hqLV9vazFFYU5vN1NqcG1hTGxaVngxM2xmNHZ4Z0lQNlFXUDRSNFhpZkVTbmdHN2JMcEZDLWE3dXJ8c8pRK4ukcoJvaFLEKUks7-a2-isGdetBvTmjaUIWs2Q="
	sessionData := `MTQ5MTI4ODQ3NHxHSThCS19qQ2FsbWJ2ZFc3aUNPSUM4RllPRVhTd1had19jS2w0MjE5WU1qLXRLempVeWNhLUFaejhvMEVyY1JmenBLSjRMYXNvd291elN5T2wtMy12dkhRWFlFRThDQmN2VTBnUWZ6UExLaW9zUFFZbnB1YV9VOXJORXNnLWtCT0pOQk5HYzhmVndpR3ZVNUZyRnh4Qy05cHdJOHRNYVJ4YXRGNEtObU94WG1iVnVZM1pJSkdERHpMbzN1VUpxVzgycnZUWWtlbnZUTWdxRDRCTEJEaEhsNHNnZmR3RFJrV1AyUkdfckNFa1lKb2d3VWR3Y0FzS1JtUllPTi0ydHQ3T2JDaUcxQ1JEQUVLbzNUNlRzM1VlUHVTYmtwWUItbFp5czRtd3FGb1VmcHFETkthR2dMWkpHRmM1a1NfZWxXLUljZUdMblJCYTZuTE12NkRvV0ZrQnVYMFFsdUM3clpFdzdUYUFIcFhSaUQ0bHZRS19ZRzExbzlLUTdCVTZnT2xNTmZIal9Oc2VOdWJtd3M3bnlibmlpLTJDRnRkQ1hyU2hYV0pienlTREl1QnRoZHNaQ3lvaGYzbWFCajA0Zi1XcFBwOXF3PT181BI_L4loH_Kcug8MEVnsFj4Ha25umy-8fI0atPVo04k=`

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	// write session file
	sessionFile := filepath.Join(getTestdataPath(), "session", m.cookieMgr.Options.Name+"_SWkGHtLck_sv7kWKDvvN8mwSq3CPfmkoRkz1POMtnx8")
	_ = ioutil.WriteFile(sessionFile, []byte(sessionData), 0600)

	header := http.Header{}
	header.Add("Cookie", cookieValue)
	req := &http.Request{
		Header: header,
	}

	session := m.GetSession(req)
	assertSessionValue(t, session)
}

func TestSessionFileStoreSave(t *testing.T) {
	testSessionStoreSave(t, `
	security {
	  session {
	    store {
	      type = "file"
	      filepath = "testdata/session"
	    }

	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)
}

func TestSessionFileStoreDeleteAndCleanup(t *testing.T) {
	sessionDir := filepath.Join(getTestdataPath(), "session")
	defer ess.DeleteFiles(sessionDir)

	m := createTestManager(t, `
	security {
	  session {
	    store {
	      type = "file"
	      filepath = "testdata/session"
	    }

	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	session := m.NewSession()
	session.Set("my-key-1", "my key value 1")
	session.Set("my-key-2", 65454523452)
	session.Set("my-key-3", map[interface{}]interface{}{"test1": "test1value", "test2": 6546546})
	session.SetFlash("my-1", "my 1 flash value")
	session.SetFlash("my-2", 364534.4637)

	w1 := httptest.NewRecorder()

	sid := session.ID
	err := m.SaveSession(w1, session)
	assert.Nil(t, err)

	result1 := w1.Result()
	assert.NotNil(t, result1)
	assert.False(t, result1.Header.Get(ahttp.HeaderSetCookie) == "")

	// Check session in store
	files, _ := ess.FilesPath(sessionDir, false)
	assert.Equal(t, 1, len(files))
	assert.True(t, m.store.IsExists(sid))

	// Cleanup run manual for test
	m.store.Cleanup(m)

	// Delete session
	session.ID = sid
	session.Clear()
	w2 := httptest.NewRecorder()
	err = m.SaveSession(w2, session)
	assert.Nil(t, err)

	result2 := w2.Result()
	assert.NotNil(t, result2)

	// Expires value
	setCookieValue := result2.Header.Get(ahttp.HeaderSetCookie)
	assert.False(t, setCookieValue == "")
	assert.True(t, strings.Contains(setCookieValue, "Thu, 01 Jan 1970 00:00:01 GMT"))

	// Sesion data should be delete here
	files, _ = ess.FilesPath(sessionDir, false)
	assert.Equal(t, 0, len(files))
	assert.False(t, m.store.IsExists(sid))
}
