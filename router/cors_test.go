// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// aahframework.org/router source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package router

import (
	"io/ioutil"
	"testing"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/test.v0/assert"
)

func TestRouterCORS1(t *testing.T) {
	_ = log.SetLevel("TRACE")
	log.SetWriter(ioutil.Discard)
	router, err := createRouter("routes-cors-1.conf")
	assert.FailNowOnError(t, err, "")

	domain := router.Lookup("localhost:8080")
	assert.True(t, domain.CORS.IsOriginAllowed("*"))
	assert.True(t, domain.CORS.IsHeadersAllowed("Accept, Origin"))
	assert.True(t, domain.CORS.IsMethodAllowed("HEAD"))
	assert.False(t, domain.CORS.AllowCredentials)
	assert.False(t, domain.CORS.IsOriginAllowed(""))

	routes := router.Lookup("localhost:8080").routes
	assert.NotNil(t, routes)
	assert.Equal(t, 8, len(routes))

	getUserRoute := routes["get_user"]
	assert.NotNil(t, getUserRoute.CORS)
	assert.True(t, getUserRoute.CORS.IsOriginAllowed("https://www.mydomain.com"))
	assert.True(t, getUserRoute.CORS.IsHeadersAllowed("X-Get-Test2"))
	assert.False(t, getUserRoute.CORS.IsHeadersAllowed("Accept"))
	assert.True(t, getUserRoute.CORS.IsMethodAllowed("DELETE"))
	assert.False(t, getUserRoute.CORS.IsMethodAllowed("HEAD"))
	assert.False(t, getUserRoute.CORS.AllowCredentials)

	updateUserRoute := routes["update_user"]
	assert.Nil(t, updateUserRoute.CORS)

	getUserSettingsRoute := routes["get_user_settings"]
	assert.True(t, getUserSettingsRoute.CORS.IsMethodAllowed("POST"))
	assert.True(t, getUserSettingsRoute.CORS.IsHeadersAllowed("Authorization"))
}

func TestRouterCORS2(t *testing.T) {
	_ = log.SetLevel("TRACE")
	log.SetWriter(ioutil.Discard)
	router, err := createRouter("routes-cors-2.conf")
	assert.FailNowOnError(t, err, "")

	domain := router.Lookup("localhost:8080")
	assert.True(t, domain.CORS.IsOriginAllowed("https://www.basemydomain.com"))
	assert.True(t, domain.CORS.IsHeadersAllowed("X-Base-Test2"))
	assert.True(t, domain.CORS.IsMethodAllowed("DELETE"))
	assert.True(t, ess.IsSliceContainsString(domain.CORS.ExposeHeaders, "X-Base-Test2"))
	assert.True(t, domain.CORS.AllowCredentials)
	assert.Equal(t, "172800", domain.CORS.MaxAge)

	routes := router.Lookup("localhost:8080").routes
	assert.NotNil(t, routes)
	assert.Equal(t, 8, len(routes))

	getUserRoute := routes["get_user"]
	assert.NotNil(t, getUserRoute.CORS)
	assert.True(t, getUserRoute.CORS.IsOriginAllowed("https://www.mydomain.com"))
	assert.True(t, getUserRoute.CORS.IsHeadersAllowed("X-Get-Test2"))
	assert.False(t, getUserRoute.CORS.IsHeadersAllowed("Accept"))
	assert.True(t, getUserRoute.CORS.IsMethodAllowed("DELETE"))
	assert.False(t, getUserRoute.CORS.IsMethodAllowed("HEAD"))
	assert.True(t, getUserRoute.CORS.AllowCredentials)
	assert.Equal(t, "86400", getUserRoute.CORS.MaxAge)

	deleteUserRoute := routes["delete_user"]
	assert.True(t, deleteUserRoute.CORS.IsOriginAllowed("https://www.basemydomain.com"))
	assert.True(t, deleteUserRoute.CORS.IsHeadersAllowed("X-Delete-Test2"))
	assert.False(t, deleteUserRoute.CORS.IsHeadersAllowed("Accept"))
	assert.True(t, deleteUserRoute.CORS.IsMethodAllowed("DELETE"))
	assert.False(t, deleteUserRoute.CORS.IsMethodAllowed("HEAD"))
	assert.True(t, deleteUserRoute.CORS.AllowCredentials)
	assert.Equal(t, "172800", deleteUserRoute.CORS.MaxAge)

	updateUserRoute := routes["update_user"]
	assert.Nil(t, updateUserRoute.CORS)

}
