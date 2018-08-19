// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"aahframework.org/config.v0"
	"aahframework.org/essentials.v0"
	"aahframework.org/security.v0/cookie"
	"aahframework.org/test.v0/assert"
)

func TestSessionEncodeAndDecodeNoSignEnc(t *testing.T) {
	m := createTestManager(t, `
		security {
			session {
				#prefix = "test-name"
			}
		}
	`)

	// string value
	// name := "test-name"
	value := "This is testing of encode and decode value without sign and encryption"

	assert.False(t, m.IsStateful())
	assert.True(t, m.IsCookieStore())

	encodedStr, err := m.Encode(value)
	assert.Nil(t, err)
	assert.False(t, encodedStr == value)

	var result string
	err = m.Decode(encodedStr, &result)
	assert.Nil(t, err)
	assert.Equal(t, value, result)

	// Object value
	session := m.NewSession()
	session.Set("my-key-1", "my key value 1")
	session.Set("my-key-2", 65454523452)
	session.Set("my-key-3", map[interface{}]interface{}{"test1": "test1value", "test2": 6546546})

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	encodedStr, err = m.Encode(session)
	assert.FailNowOnError(t, err, "unexpected")
	assert.Nil(t, err)
	assert.False(t, encodedStr == "")

	var resultSession Session
	err = m.Decode(encodedStr, &resultSession)
	assert.Nil(t, err)
	assertSessionValue(t, &resultSession)
	ReleaseSession(&resultSession)
}

func TestSessionEncodeAndDecodeWithSignEnc(t *testing.T) {
	m := createTestManager(t, `
	security {
	  session {
	    ttl = "30m"
	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	session := m.NewSession()
	session.Set("my-key-1", "my key value 1")
	session.Set("my-key-2", 65454523452)
	session.Set("my-key-3", map[interface{}]interface{}{"test1": "test1value", "test2": 6546546})

	// register custom type
	gob.Register(map[interface{}]interface{}{})

	encodedStr, err := m.Encode(session)
	assert.FailNowOnError(t, err, "unexpected")
	assert.Nil(t, err)
	assert.False(t, encodedStr == "")

	var resultSession Session
	err = m.Decode(encodedStr, &resultSession)
	assert.Nil(t, err)
	assertSessionValue(t, &resultSession)
	ReleaseSession(&resultSession)
}

func TestSessionRegisterStore(t *testing.T) {
	err := AddStore("file", &FileStore{})
	assert.NotNil(t, err)
	assert.Equal(t, "session: store name 'file' is already added, skip it", err.Error())

	err = AddStore("custom", nil)
	assert.NotNil(t, err)
	assert.Equal(t, errors.New("security/session: store value is nil"), err)
}

func TestSessionStoreNotExists(t *testing.T) {
	cfg, _ := config.ParseString(`
	security {
	  session {
	    store {
	      type = "custom"
	    }
	  }
	}
  `)
	m, err := NewManager(cfg)
	assert.NotNil(t, err)
	assert.Nil(t, m)
	assert.Equal(t, "session: store name 'custom' not exists", err.Error())
}

func TestSessionManagerMisc(t *testing.T) {
	m := createTestManager(t, `
	security {
	  session {
	    ttl = "30m"
	    sign_key = "eFWLXEewECptbDVXExokRTLONWxrTjfV"
	    enc_key = "KYqklJsgeclPpZutTeQKNOTWlpksRBwA"
	  }
	}
  `)

	session, err := m.DecodeToSession("MTQ5MTM2OTI4NXxpV1l2SHZrc0tZaXprdlA5Ql9ZS3RWOC1yOFVoWElack1VTGJIM01aV2dGdmJvamJOR2Rmc05KQW1SeHNTS2FoNEJLY2NFN2MyenVCbGllaU1NRFV88hn8MIb0L5HFU6GAkvwYjQ1rvmaL3lG3am2ZageHxQ0=")
	assert.Nil(t, session)
	assert.True(t, err == cookie.ErrCookieTimestampIsExpired)

	str, err := m.DecodeToString("Expecting base64 decode error")
	assert.True(t, err == ess.ErrBase64Decode)
	assert.True(t, str == "")

	result, err := toBytes([]byte("bytes test"))
	assert.Nil(t, err)
	assert.Equal(t, "bytes test", string(result))

	_, err = toSeconds("40s")
	assert.NotNil(t, err)
	assert.Equal(t, "unsupported time unit '40s' on 'session.ttl'", err.Error())

	_, err = toSeconds("sm")
	assert.NotNil(t, err)
	assert.Equal(t, "time: invalid duration sm", err.Error())

	es := m.NewSession()
	es.Set("my-key-1", "my key value 1")
	es.Set("my-key-2", 65454523452)
	es.Set("my-key-3", map[interface{}]string{"test1": "test1value", "test2": "6546546"})
	_, err = encodeGob(es)
	assert.NotNil(t, err)
	assert.Equal(t, "gob: type not registered for interface: map[interface {}]string", err.Error())
	assert.Equal(t, 0, es.GetInt("not-exists"))
	assert.Equal(t, int64(0), es.GetInt64("not-exists"))
	assert.Equal(t, "", es.GetString("not-exists"))
	assert.False(t, es.GetBool("not-exists"))
	assert.Equal(t, float32(0.0), es.GetFloat32("not-exists"))
	assert.Equal(t, float64(0.0), es.GetFloat64("not-exists"))
}

func assertSessionValue(t *testing.T, s *Session) {
	t.Logf("Session: %v", s)
	assert.NotNil(t, s)
	assert.Equal(t, "my key value 1", s.GetString("my-key-1"))
	assert.Equal(t, 65454523452, s.GetInt("my-key-2"))
	assert.Nil(t, s.Get("not-exists"))

	maps := s.Get("my-key-3").(map[interface{}]interface{})
	assert.Equal(t, "test1value", maps["test1"])
	assert.Equal(t, 6546546, maps["test2"])
}

func createTestManager(t *testing.T, cfgStr string) *Manager {
	cfg, _ := config.ParseString(cfgStr)
	m, err := NewManager(cfg)
	assert.FailNowOnError(t, err, "unexpected")
	return m
}

func getTestdataPath() string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "testdata")
}
