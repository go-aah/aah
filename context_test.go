// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"aahframework.org/ahttp.v0"
	"aahframework.org/config.v0"
	"aahframework.org/router.v0"
	"aahframework.org/test.v0/assert"
)

type (
	Anonymous1 struct {
		Name string
	}

	Func1 func(e *Event)

	Level1 struct{ *Context }

	Level2 struct{ Level1 }

	Level3 struct{ Level2 }

	Level4 struct{ Level3 }

	Path1 struct {
		Anonymous Anonymous1
		*Context
	}

	Path2 struct {
		Level1
		Path1
		Level4
		Func1
	}
)

func TestContextReverseURL(t *testing.T) {
	appCfg, _ := config.ParseString("")
	err := initRoutes(getTestdataPath(), appCfg)
	assert.Nil(t, err)
	assert.NotNil(t, AppRouter())

	ctx := &Context{
		Req: getAahRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", ""),
	}

	reverseURL := ctx.ReverseURL("version_home", "v0.1")
	assert.Equal(t, "//localhost:8080/doc/v0.1", reverseURL)

	reverseURL = ctx.ReverseURLm("show_doc", map[string]interface{}{
		"version": "v0.2",
		"content": "getting-started.html",
	})
	assert.Equal(t, "//localhost:8080/doc/v0.2/getting-started.html", reverseURL)

	ctx.Reset()
}

func TestContextViewArgs(t *testing.T) {
	ctx := &Context{viewArgs: make(map[string]interface{}, 0)}

	ctx.AddViewArg("key1", "key1 value")
	assert.Equal(t, "key1 value", ctx.viewArgs["key1"])
	assert.Nil(t, ctx.viewArgs["notexists"])
}

func TestContextMsg(t *testing.T) {
	i18nDir := filepath.Join(getTestdataPath(), appI18nDir())
	err := initI18n(i18nDir)
	assert.Nil(t, err)
	assert.NotNil(t, AppI18n())

	ctx := &Context{
		Req: getAahRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", "en-us;q=0.0,en;q=0.7, da, en-gb;q=0.8"),
	}

	msg := ctx.Msg("label.pages.site.get_involved.title")
	assert.Equal(t, "", msg)

	msg = ctx.Msgl(ahttp.ToLocale(&ahttp.AcceptSpec{Value: "en", Raw: "en"}), "label.pages.site.get_involved.title")
	assert.Equal(t, "en: Get Involved - aah web framework for Go", msg)

	ctx.Req = getAahRequest("GET", "http://localhost:8080/doc/v0.3/mydoc.html", "en-us;q=0.0,en;q=0.7,en-gb;q=0.8")
	msg = ctx.Msg("label.pages.site.get_involved.title")
	assert.Equal(t, "en: Get Involved - aah web framework for Go", msg)

	ctx.Reset()
}

func TestContextSetTarget(t *testing.T) {
	addToCRegistry()

	ctx := &Context{}

	err1 := ctx.setTarget(&router.Route{Controller: "Level3", Action: "Testing"})
	assert.Nil(t, err1)
	assert.Equal(t, "Level3", ctx.controller)
	assert.NotNil(t, ctx.action)
	assert.Equal(t, "Testing", ctx.action.Name)
	assert.NotNil(t, ctx.action.Parameters)
	assert.Equal(t, "userId", ctx.action.Parameters[0].Name)

	err2 := ctx.setTarget(&router.Route{Controller: "NoController"})
	assert.Equal(t, errTargetNotFound, err2)

	err3 := ctx.setTarget(&router.Route{Controller: "Level3", Action: "NoAction"})
	assert.Equal(t, errTargetNotFound, err3)
}

func TestContextSession(t *testing.T) {
	cfgDir := filepath.Join(getTestdataPath(), appConfigDir())
	err := initConfig(cfgDir)
	assert.Nil(t, err)

	err = initSecurity(cfgDir, AppConfig())
	assert.Nil(t, err)

	ctx := &Context{viewArgs: make(map[string]interface{})}
	s1 := ctx.Session()
	assert.NotNil(t, s1)
	assert.True(t, s1.IsNew)
	assert.NotNil(t, s1.ID)
}

func TestContextCookie(t *testing.T) {
	cookieValue := `aah_session=MTQ5MTI4ODQ3NHxHSThCS19qQ2FsbWJ2ZFc3aUNPSUM4RllPRVhTd1had19jS2w0MjE5WU1qLXRLempVeWNhLUFaejhvMEVyY1JmenBLSjRMYXNvd291elN5T2wtMy12dkhRWFlFRThDQmN2VTBnUWZ6UExLaW9zUFFZbnB1YV9VOXJORXNnLWtCT0pOQk5HYzhmVndpR3ZVNUZyRnh4Qy05cHdJOHRNYVJ4YXRGNEtObU94WG1iVnVZM1pJSkdERHpMbzN1VUpxVzgycnZUWWtlbnZUTWdxRDRCTEJEaEhsNHNnZmR3RFJrV1AyUkdfckNFa1lKb2d3VWR3Y0FzS1JtUllPTi0ydHQ3T2JDaUcxQ1JEQUVLbzNUNlRzM1VlUHVTYmtwWUItbFp5czRtd3FGb1VmcHFETkthR2dMWkpHRmM1a1NfZWxXLUljZUdMblJCYTZuTE12NkRvV0ZrQnVYMFFsdUM3clpFdzdUYUFIcFhSaUQ0bHZRS19ZRzExbzlLUTdCVTZnT2xNTmZIal9Oc2VOdWJtd3M3bnlibmlpLTJDRnRkQ1hyU2hYV0pienlTREl1QnRoZHNaQ3lvaGYzbWFCajA0Zi1XcFBwOXF3PT181BI_L4loH_Kcug8MEVnsFj4Ha25umy-8fI0atPVo04k=`

	req, _ := http.NewRequest("GET", "http://localhost:8080/user/registration", nil)
	req.Header.Add(ahttp.HeaderCookie, cookieValue)

	aahReq := ahttp.ParseRequest(req, &ahttp.Request{})
	ctx := &Context{Req: aahReq}

	v1, err := ctx.Cookie("aah_session")
	assert.Nil(t, err)
	assert.NotNil(t, v1)
	assert.Equal(t, "aah_session", v1.Name)

	v2 := ctx.Cookies()
	assert.NotNil(t, v1)
	assert.True(t, len(v2) == 1)
}

func TestContextAbort(t *testing.T) {
	ctx := &Context{}

	assert.False(t, ctx.abort)
	ctx.Abort()
	assert.True(t, ctx.abort)
}

func TestContextNil(t *testing.T) {
	ctx := &Context{}

	assert.Nil(t, ctx.Reply())
	assert.Nil(t, ctx.ViewArgs())
}

func TestContextEmbeddedAndController(t *testing.T) {
	addToCRegistry()

	assertEmbeddedIndexes(t, Level1{}, [][]int{{0}})
	assertEmbeddedIndexes(t, Level2{}, [][]int{{0, 0}})
	assertEmbeddedIndexes(t, Level3{}, [][]int{{0, 0, 0}})
	assertEmbeddedIndexes(t, Level4{}, [][]int{{0, 0, 0, 0}})
	assertEmbeddedIndexes(t, Path1{}, [][]int{{1}})
	assertEmbeddedIndexes(t, Path2{}, [][]int{{0, 0}, {1, 1}, {2, 0, 0, 0, 0}})
}

func TestContextSetURL(t *testing.T) {
	ctx := &Context{
		Req: getAahRequest("POST", "http://localhost:8080/users/edit", ""),
	}

	assert.Equal(t, "localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method)
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.False(t, ctx.decorated)

	// No effects, since decorated is false
	ctx.SetURL("http://status.localhost:8080/maintenance")
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.Equal(t, "localhost:8080", ctx.Req.Host)

	// now it affects
	ctx.decorated = true
	ctx.SetURL("http://status.localhost:8080/maintenance")
	assert.True(t, ctx.decorated)
	assert.Equal(t, "status.localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method) // no change expected
	assert.Equal(t, "/maintenance", ctx.Req.Path)

	// incorrect URL
	ctx.SetURL("http://status. localhost :8080//maintenance")
	assert.Equal(t, "status.localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method) // no change expected
	assert.Equal(t, "/maintenance", ctx.Req.Path)
}

func TestContextSetMethod(t *testing.T) {
	ctx := &Context{
		Req: getAahRequest("POST", "http://localhost:8080/users/edit", ""),
	}

	assert.Equal(t, "localhost:8080", ctx.Req.Host)
	assert.Equal(t, "POST", ctx.Req.Method)
	assert.Equal(t, "/users/edit", ctx.Req.Path)
	assert.False(t, ctx.decorated)

	// No effects, since decorated is false
	ctx.SetMethod("GET")
	assert.Equal(t, "POST", ctx.Req.Method)

	// now it affects
	ctx.decorated = true
	ctx.SetMethod("get")
	assert.Equal(t, "GET", ctx.Req.Method)
	assert.Equal(t, "localhost:8080", ctx.Req.Host) // no change expected
	assert.Equal(t, "/users/edit", ctx.Req.Path)    // no change expected
	assert.True(t, ctx.decorated)

	// invalid method
	ctx.SetMethod("nomethod")
	assert.Equal(t, "GET", ctx.Req.Method)
}

func assertEmbeddedIndexes(t *testing.T, c interface{}, expected [][]int) {
	actual := findEmbeddedContext(reflect.TypeOf(c))
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Indexes do not match. expected %v actual %v", expected, actual)
	}
}

func addToCRegistry() {
	cRegistry = controllerRegistry{}

	AddController((*Level1)(nil), []*MethodInfo{
		{
			Name:       "Index",
			Parameters: []*ParameterInfo{},
		},
	})
	AddController((*Level2)(nil), []*MethodInfo{
		{
			Name:       "Scope",
			Parameters: []*ParameterInfo{},
		},
	})
	AddController((*Level3)(nil), []*MethodInfo{
		{
			Name: "Testing",
			Parameters: []*ParameterInfo{
				{
					Name: "userId",
					Type: reflect.TypeOf((*int)(nil)),
				},
			},
		},
	})
	AddController((*Level4)(nil), nil)
	AddController((*Path1)(nil), nil)
	AddController((*Path2)(nil), nil)
}

func getAahRequest(method, target, al string) *ahttp.Request {
	rawReq := httptest.NewRequest(method, target, nil)
	rawReq.Header.Add(ahttp.HeaderAcceptLanguage, al)
	return ahttp.ParseRequest(rawReq, &ahttp.Request{})
}

func getTestdataPath() string {
	return filepath.Join(getWorkingDir(), "testdata")
}
