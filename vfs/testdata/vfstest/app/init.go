// aah application initialization - configuration, server extensions, middleware's, etc.
// Customize it per your application needs.

package main

import (

	"aahframe.work/aah/aah.v0"

	// Registering HTML minifier for web application
	_ "github.com/aah-cb/minify"
)

func init() {

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Server Extensions
	// Doc: https://docs.aahframework.org/server-extension.html
	//
	// Best Practice: Define a function with meaningful name in a package and
	// register it here. Extensions function name gets logged in the log,
	// its very helpful to have meaningful log information.
	//
	// Such as:
	//    - Dedicated package for config loading
	//    - Dedicated package for datasource connections
	//    - etc
	//__________________________________________________________________________

	// Event: OnInit
	//
	// Published right after the `aah.AppConfig()` is loaded.
	//
	// aah.OnInit(config.LoadRemote)

	// Event: OnStart
	//
	// Published right before the start of aah server.
	//
	// aah.OnStart(db.Connect)
	// aah.OnStart(cache.Load)
	aah.OnStart(SubscribeHTTPEvents)

	// Event: OnShutdown
	//
	// Published on receiving OS Signals `SIGINT` or `SIGTERM`.
	//
	// aah.OnShutdown(cache.Flush)
	// aah.OnShutdown(db.Disconnect)

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Middleware's
	// Doc: https://docs.aahframework.org/middleware.html
	//
	// Executed in the order they are defined. It is recommended; NOT to change
	// the order of pre-defined aah framework middleware's.
	//__________________________________________________________________________
	aah.Middlewares(
		aah.RouteMiddleware,
		aah.CORSMiddleware,
		aah.BindMiddleware,
		aah.AntiCSRFMiddleware,
		aah.AuthcAuthzMiddleware,

		//
		// NOTE: Register your Custom middleware's right here
		//

		aah.ActionMiddleware,
	)

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Add Application Error Handler
	// Doc: https://docs.aahframework.org/error-handling.html
	//__________________________________________________________________________
	// aah.SetErrorHandler(AppErrorHandler)

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Add Custom Template Functions
	// Doc: https://docs.aahframework.org/template-funcs.html
	//__________________________________________________________________________
	// aah.AddTemplateFunc(template.FuncMap{
	// // ...
	// })

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Add Custom Session Store
	// Doc: https://docs.aahframework.org/session.html
	//__________________________________________________________________________
	// aah.AddSessionStore("redis", &RedisSessionStore{})

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Add Custom value Parser
	// Doc: https://docs.aahframework.org/request-parameters-auto-bind.html
	//__________________________________________________________________________
	// if err := aah.AddValueParser(reflect.TypeOf(CustomType{}), customParser); err != nil {
	//   log.Error(err)
	// }

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// Add Custom Validation Functions
	// Doc: https://godoc.org/gopkg.in/go-playground/validator.v9
	//__________________________________________________________________________
	// Obtain aah validator instance, then add yours
	// validator := aah.Validator()
	//
	// // Add your validation funcs

}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// HTTP Events
//
// Subscribing HTTP events on app start.
//__________________________________________________________________________

func SubscribeHTTPEvents(_ *aah.Event) {
	// he := aah.AppHTTPEngine()

	// Event: OnRequest
	// he.OnRequest(myserverext.OnRequest)

	// Event: OnPreReply
	// he.OnPreReply(myserverext.OnPreReply)

	// Event: OnPostReply
	// he.OnPostReply(myserverext.OnPostReply)

	// Event: OnPreAuth
	// he.OnPreAuth(myserverext.OnPreAuth)

	// Event: OnPostAuth
	// Published right after the successful Authentication
	// he.OnPostAuth(myserverext.PostAuthEvent)
}

