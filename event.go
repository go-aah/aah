// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"sort"
	"sync"

	"aahframework.org/log.v0"
)

const (
	// EventOnInit event is fired right after aah application config is initialized.
	EventOnInit = "OnInit"

	// EventOnStart event is fired before HTTP/Unix listener starts
	EventOnStart = "OnStart"

	// EventOnShutdown event is fired when server recevies an interrupt or kill command.
	EventOnShutdown = "OnShutdown"

	//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
	// HTTP Engine events
	//______________________________________________________________________________

	// EventOnRequest event is fired when server recevies an incoming request.
	EventOnRequest = "OnRequest"

	// EventOnPreReply event is fired when before server writes the reply on the wire.
	// Except when
	//   1) `Reply().Done()`,
	//   2) `Reply().Redirect(...)` is called.
	// Refer `aah.Reply.Done()` godoc for more info.
	EventOnPreReply = "OnPreReply"

	// EventOnPostReply event is fired when before server writes the reply on the wire.
	// Except when
	//   1) `Reply().Done()`,
	//   2) `Reply().Redirect(...)` is called.
	// Refer `aah.Reply.Done()` godoc for more info.
	EventOnPostReply = "OnPostReply"

	// EventOnAfterReply DEPRECATED use EventOnPostReply instead.
	//
	// Note: DEPRECATED elements to be removed in `v1.0.0` release.
	EventOnAfterReply = EventOnPostReply

	// EventOnPreAuth event is fired before server Authenticates & Authorizes an incoming request.
	EventOnPreAuth = "OnPreAuth"

	// EventOnPostAuth event is fired after server Authenticates & Authorizes an incoming request.
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
	a.eventStore.Subscribe(EventOnInit, EventCallback{
		Callback: ecb,
		CallOnce: true,
		priority: parsePriority(priority...),
	})
}

func (a *app) OnStart(ecb EventCallbackFunc, priority ...int) {
	a.eventStore.Subscribe(EventOnStart, EventCallback{
		Callback: ecb,
		CallOnce: true,
		priority: parsePriority(priority...),
	})
}

func (a *app) OnShutdown(ecb EventCallbackFunc, priority ...int) {
	a.eventStore.Subscribe(EventOnShutdown, EventCallback{
		Callback: ecb,
		CallOnce: true,
		priority: parsePriority(priority...),
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

// DEPRECATED: use SubscribeEventFunc instead.
func (a *app) SubscribeEventf(eventName string, ecf EventCallbackFunc) {
	a.showDeprecatedMsg("SubscribeEventf, use 'SubscribeEventFunc' instead")
	a.SubscribeEventFunc(eventName, ecf)
}

func (a *app) UnsubscribeEvent(eventName string, ec EventCallback) {
	a.UnsubscribeEventFunc(eventName, ec.Callback)
}

func (a *app) UnsubscribeEventFunc(eventName string, ecf EventCallbackFunc) {
	a.eventStore.Unsubscribe(eventName, ecf)
}

// DEPRECATED: use UnsubscribeEventFunc instead.
func (a *app) UnsubscribeEventf(eventName string, ecf EventCallbackFunc) {
	a.showDeprecatedMsg("UnsubscribeEventf, use 'UnsubscribeEventFunc' instead")
	a.UnsubscribeEventFunc(eventName, ecf)
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
	mu          *sync.Mutex
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
		if funcEqual(ec.Callback, callback) {
			es.subscribers[event] = append(es.subscribers[event][:idx], es.subscribers[event][idx+1:]...)
			es.a.Log().Debugf("Callback: %s, unsubscribed from event: %s", funcName(callback), event)
			return
		}
	}

	es.a.Log().Warnf("Given callback: %s, not found in eventStore for event: %s", funcName(callback), event)
}

// SubscriberCount method returns subscriber count for given event name.
func (es *EventStore) SubscriberCount(eventName string) int {
	if subs, found := es.subscribers[eventName]; found {
		return len(subs)
	}
	return 0
}

func (es *EventStore) sortAndPublishSync(e *Event) {
	if es.IsEventExists(e.Name) {
		sort.Sort(es.subscribers[e.Name])
		es.PublishSync(e)
	}
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// EventCallbacks methods
//______________________________________________________________________________

// Sort interface for EventCallbacks
func (ec EventCallbacks) Len() int           { return len(ec) }
func (ec EventCallbacks) Less(i, j int) bool { return ec[i].priority < ec[j].priority }
func (ec EventCallbacks) Swap(i, j int)      { ec[i], ec[j] = ec[j], ec[i] }
