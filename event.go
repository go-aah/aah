// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// Source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"sort"
	"sync"

	"aahframe.work/essentials"
	"aahframe.work/internal/util"
	"aahframe.work/log"
)

const (
	// EventOnInit is published once the aah.AppConfig() is loaded. At this stage,
	// only aah.conf config is initialized. App Variables, Routes, i18n, Security,
	// View Engine, Logs and so on will be initialized after this event.
	EventOnInit = "OnInit"

	// EventOnStart is published just before the start of aah Server.
	// The application is completely initialized at this stage. The server
	// is yet to be started.
	EventOnStart = "OnStart"

	// EventOnPreShutdown is published when application receives OS Signals
	// `SIGINT` or `SIGTERM` and before the triggering graceful shutdown. After this
	// event, aah triggers graceful shutdown with config value of
	// `server.timeout.grace_shutdown`.
	EventOnPreShutdown = "OnPreShutdown"

	// EventOnPostShutdown is published just after the successful grace shutdown
	// of aah server and then application does clean exit.
	EventOnPostShutdown = "OnPostShutdown"

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// HTTP Engine events
	//______________________________________________________________________________

	// EventOnRequest is published on each incoming request to the aah server.
	EventOnRequest = "OnRequest"

	// EventOnPreReply is published just before writing a reply/response on the wire.
	// At this point, the response writer is clean. i.e. Headers, Cookies, Redirects,
	// Status Code and Response Body are not written. event is published when
	// before server writes the reply on the wire.
	//
	// Except when
	//
	//   1) `Reply().Done()`,
	//
	//   2) `Reply().Redirect(...)` is called.
	//
	// Refer `aah.Reply().Done()` godoc for more info.
	EventOnPreReply = "OnPreReply"

	// EventOnHeaderReply is published before writing HTTP header Status.
	// At this point, all the headers except the header Status get written on
	// the http.ResponseWriter.
	//
	// Except when
	//
	//   1) `Reply().Done()`,
	//
	//   2) `Reply().Redirect(...)` is called.
	//
	// Refer `aah.Reply().Done()` godoc for more info.
	EventOnHeaderReply = "OnHeaderReply"

	// EventOnPostReply is published right after the response gets written on the
	// wire. We can do nothing about the response, however the context has valuable
	// information such as response bytes size, response status code, etc.
	//
	// Except when
	//
	//   1) `Reply().Done()`,
	//
	//   2) `Reply().Redirect(...)` is called.
	//
	// Refer `aah.Reply().Done()` godoc for more info.
	EventOnPostReply = "OnPostReply"

	// EventOnPreAuth is published just before the Authentication and Authorization.
	EventOnPreAuth = "OnPreAuth"

	// EventOnPostAuth is published once the Authentication and Authorization
	// info gets populated into Subject.
	EventOnPostAuth = "OnPostAuth"
)

type (
	// Event type holds the details of single event.
	Event struct {
		Name string
		Data interface{}
	}

	// EventCallback type is store particular callback in priority for calling sequance.
	EventCallback struct {
		Callback EventCallbackFunc
		CallOnce bool

		published bool
		priority  int
	}

	// EventCallbacks type is slice of `EventCallback` type.
	EventCallbacks []EventCallback

	// EventCallbackFunc is signature of event callback function.
	EventCallbackFunc func(e *Event)
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app event methods
//______________________________________________________________________________

func (a *app) OnInit(ecb EventCallbackFunc, priority ...int) {
	a.subcribeAppEvent(EventOnInit, ecb, priority)
}

func (a *app) OnStart(ecb EventCallbackFunc, priority ...int) {
	a.subcribeAppEvent(EventOnStart, ecb, priority)
}

func (a *app) OnPreShutdown(ecb EventCallbackFunc, priority ...int) {
	a.subcribeAppEvent(EventOnPreShutdown, ecb, priority)
}

func (a *app) OnPostShutdown(ecb EventCallbackFunc, priority ...int) {
	a.subcribeAppEvent(EventOnPostShutdown, ecb, priority)
}

func (a *app) subcribeAppEvent(eventName string, ecb EventCallbackFunc, priority []int) {
	a.SubscribeEvent(eventName, EventCallback{
		Callback: ecb,
		CallOnce: true,
		priority: parsePriority(priority),
	})
}

func (a *app) PublishEvent(eventName string, data interface{}) {
	a.eventStore.Publish(&Event{Name: eventName, Data: data})
}

func (a *app) PublishEventSync(eventName string, data interface{}) {
	a.eventStore.PublishSync(&Event{Name: eventName, Data: data})
}

func (a *app) SubscribeEvent(eventName string, ec EventCallback) {
	a.eventStore.Subscribe(eventName, ec)
}

func (a *app) SubscribeEventFunc(eventName string, ecf EventCallbackFunc) {
	a.eventStore.Subscribe(eventName, EventCallback{Callback: ecf})
}

func (a *app) UnsubscribeEvent(eventName string, ec EventCallback) {
	a.UnsubscribeEventFunc(eventName, ec.Callback)
}

func (a *app) UnsubscribeEventFunc(eventName string, ecf EventCallbackFunc) {
	a.eventStore.Unsubscribe(eventName, ecf)
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// app methods
//______________________________________________________________________________

func (a *app) EventStore() *EventStore {
	return a.eventStore
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// EventStore
//______________________________________________________________________________

// EventStore type holds all the events belongs to aah application.
type EventStore struct {
	a           *app
	mu          sync.RWMutex
	subscribers map[string]EventCallbacks
}

// IsEventExists method returns true if given event is exists in the event store
// otherwise false.
func (es *EventStore) IsEventExists(eventName string) bool {
	_, found := es.subscribers[eventName]
	return found
}

// Publish method publishes events to subscribed callbacks asynchronously. It
// means each subscribed callback executed via goroutine.
func (es *EventStore) Publish(e *Event) {
	if !es.IsEventExists(e.Name) {
		return
	}

	es.a.Log().Debugf("Event [%s] published in asynchronous mode", e.Name)
	for idx, ec := range es.subscribers[e.Name] {
		if ec.CallOnce {
			if !ec.published {
				go func(event *Event, ecb EventCallbackFunc) {
					ecb(event)
				}(e, ec.Callback)
				es.mu.Lock()
				es.subscribers[e.Name][idx].published = true
				es.mu.Unlock()
			}
			continue
		}

		go func(event *Event, ecb EventCallbackFunc) {
			ecb(event)
		}(e, ec.Callback)
	}
}

// PublishSync method publishes events to subscribed callbacks synchronously.
func (es *EventStore) PublishSync(e *Event) {
	if !es.IsEventExists(e.Name) {
		return
	}

	if es.a.Log() == nil {
		log.Debugf("Event [%s] publishing in synchronous mode", e.Name)
	} else {
		es.a.Log().Debugf("Event [%s] publishing in synchronous mode", e.Name)
	}

	for idx, ec := range es.subscribers[e.Name] {
		if ec.CallOnce {
			if !ec.published {
				ec.Callback(e)
				es.mu.Lock()
				es.subscribers[e.Name][idx].published = true
				es.mu.Unlock()
			}
			continue
		}

		ec.Callback(e)
	}
}

// Subscribe method is to subscribe any event with event callback info.
func (es *EventStore) Subscribe(event string, ec EventCallback) {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.IsEventExists(event) {
		es.subscribers[event] = append(es.subscribers[event], ec)
		return
	}

	es.subscribers[event] = EventCallbacks{}
	es.subscribers[event] = append(es.subscribers[event], ec)
}

// Unsubscribe method is to unsubscribe any callback from event store by event.
func (es *EventStore) Unsubscribe(event string, callback EventCallbackFunc) {
	es.mu.Lock()
	defer es.mu.Unlock()
	if !es.IsEventExists(event) {
		es.a.Log().Warnf("Subscribers not exists for event: %s", event)
		return
	}

	for idx := len(es.subscribers[event]) - 1; idx >= 0; idx-- {
		ec := es.subscribers[event][idx]
		if util.FuncEqual(ec.Callback, callback) {
			es.subscribers[event] = append(es.subscribers[event][:idx], es.subscribers[event][idx+1:]...)
			es.a.Log().Debugf("Callback: %s, unsubscribed from event: %s", ess.GetFunctionInfo(callback).QualifiedName, event)
			return
		}
	}

	es.a.Log().Warnf("Given callback: %s, not found in eventStore for event: %s", ess.GetFunctionInfo(callback).QualifiedName, event)
}

// SubscriberCount method returns subscriber count for given event name.
func (es *EventStore) SubscriberCount(eventName string) int {
	if subs, found := es.subscribers[eventName]; found {
		return len(subs)
	}
	return 0
}

func (es *EventStore) sortEventSubscribers(eventName string) {
	if es.IsEventExists(eventName) {
		ec := es.subscribers[eventName]
		sort.Slice(ec, func(i, j int) bool { return ec[i].priority < ec[j].priority })
	}
}

func (es *EventStore) sortAndPublishSync(e *Event) {
	es.sortEventSubscribers(e.Name)
	es.PublishSync(e)
}

func parsePriority(priority []int) int {
	pr := 1 // default priority is 1
	if len(priority) > 0 && priority[0] > 0 {
		pr = priority[0]
	}
	return pr
}
