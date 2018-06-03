// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/security source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package session

import (
	"fmt"
	"time"
)

const flashKeyPrefix = "_flash_"

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Session
//___________________________________

// Session hold the information for particular HTTP request.
type Session struct {
	// ID method return session ID. It is dynamically generated while new session
	// creation. ID length is 32.
	//
	//Note: Do not use this value for any/derving user relation, not recommended.
	ID string

	// Values is values that stored in session object.
	Values map[string]interface{}

	// IsNew indicates whether sesison is newly created or restore from the
	// request which was already created.
	IsNew bool

	// IsAuthenticated is helpful to identify user session already authenicated or
	// not. Don't forget to set it true after successful authentication.
	IsAuthenticated bool

	// CreatedTime is when the session was created.
	CreatedTime *time.Time

	maxAge int
}

// Get method returns the value for given key otherwise nil.
func (s *Session) Get(key string) interface{} {
	if v, found := s.Values[key]; found {
		return v
	}
	return nil
}

// Set method set the value for the given key, if key already exists it updates
// the value.
//
// Note: For any complex/custom structure you would like to store in session.
// Please register those types using `gob.Register(...)`.
func (s *Session) Set(key string, value interface{}) {
	s.Values[key] = value
}

// Del method deletes the value for the given key if exists.
func (s *Session) Del(key string) {
	if s.IsKeyExists(key) {
		delete(s.Values, key)
	}
}

// IsKeyExists method returns true if given key is exists in session object
// otherwise false.
func (s *Session) IsKeyExists(key string) bool {
	_, found := s.Values[key]
	return found
}

// Clear method marks the session for deletion. It triggers the deletion at the
// end of the request for cookie and session store data.
func (s *Session) Clear() {
	s.maxAge = -1
}

// GetFlash method returns the flash messages from the session object and
// deletes it from session.
func (s *Session) GetFlash(key string) interface{} {
	key = flashKeyPrefix + key
	v := s.Get(key)
	if v != nil {
		s.Del(key)
	}
	return v
}

// SetFlash method adds flash message into session object.
func (s *Session) SetFlash(key string, value interface{}) {
	key = flashKeyPrefix + key
	s.Set(key, value)
}

// GetBool method returns the `bool` value from otherwise false.
func (s *Session) GetBool(key string) bool {
	if value := s.Get(key); value != nil {
		return value.(bool)
	}
	return false
}

// GetInt method returns the `int` value from session otherwise 0.
func (s *Session) GetInt(key string) int {
	if value := s.Get(key); value != nil {
		return value.(int)
	}
	return 0
}

// GetInt64 method returns the `int64` value from session otherwise 0.
func (s *Session) GetInt64(key string) int64 {
	if value := s.Get(key); value != nil {
		return value.(int64)
	}
	return 0
}

// GetString method returns the `string` value from session otherwise empty string.
func (s *Session) GetString(key string) string {
	if value := s.Get(key); value != nil {
		return value.(string)
	}
	return ""
}

// GetFloat32 method returns the `float32` value from session otherwise 0.
func (s *Session) GetFloat32(key string) float32 {
	if value := s.Get(key); value != nil {
		return value.(float32)
	}
	return 0
}

// GetFloat64 method returns the `float64` value from session otherwise 0.
func (s *Session) GetFloat64(key string) float64 {
	if value := s.Get(key); value != nil {
		return value.(float64)
	}
	return 0
}

// String method is stringer interface implementation.
func (s Session) String() string {
	return fmt.Sprintf("session(id:%s createdat:%s isnew:%v isauthenticated:%v values:%v)",
		s.ID, s.CreatedTime, s.IsNew, s.IsAuthenticated, s.Values)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

// Reset method resets the instance values for repurpose.
func (s *Session) Reset() {
	s.ID = ""
	s.Values = make(map[string]interface{})
	s.IsNew = false
	s.CreatedTime = nil
	s.IsAuthenticated = false
	s.maxAge = 0
}
