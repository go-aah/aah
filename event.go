// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"reflect"
	"sort"
	"sync"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

const (
	// EventOnInit event is fired right after aah application config is initialized.
	EventOnInit = "OnInit"

	// EventOnStart event is fired before HTTP/Unix listener starts
	EventOnStart = "OnStart"

	// EventOnShutdown event is fired when server recevies interrupt or kill command.
	EventOnShutdown = "OnShutdown"

	// EventOnRequest event is fired when server recevies incoming request.
	EventOnRequest = "OnRequest"

	// EventOnPreReply event is fired when before server writes the reply on the wire.
	// Except when 1) Static file request, 2) `Reply().Done()`
	// 3) `Reply().Redirect(...)` is called. Refer `aah.Reply.Done()` godoc for more info.
	EventOnPreReply = "OnPreReply"

	// EventOnAfterReply event is fired when before server writes the reply on the wire.
	// Except when 1) Static file request, 2) `Reply().Done()`
	// 3) `Reply().Redirect(...)` is called. Refer `aah.Reply.Done()` godoc for more info.
	EventOnAfterReply = "OnAfterReply"
)

var (
	appEventStore    = &EventStore{subscribers: make(map[string]EventCallbacks), mu: &sync.Mutex{}}
	onRequestFunc    EventCallbackFunc
	onPreReplyFunc   EventCallbackFunc
	onAfterReplyFunc EventCallbackFunc
)

type (
	// Event type holds the details of event generated.
	Event struct {
		Name string
		Data interface{}
	}

	// EventStore type holds all the events belongs to aah application.
	EventStore struct {
		subscribers map[string]EventCallbacks
		mu          *sync.Mutex
	}

	// EventCallback type is store particular callback in priority for calling sequance.
	EventCallback struct {
		Callback    EventCallbackFunc
		PublishOnce bool
		priority    int
		published   bool
	}

	// EventCallbacks type is slice of `EventCallback` type.
	EventCallbacks []EventCallback

	// EventCallbackFunc is signature of event callback function.
	EventCallbackFunc func(e *Event)
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Global methods
//___________________________________

// AppEventStore method returns aah application event store.
func AppEventStore() *EventStore {
	return appEventStore
}

// PublishEvent method publishes events to subscribed callbacks asynchronously. It
// means each subscribed callback executed via goroutine.
func PublishEvent(eventName string, data interface{}) {
	AppEventStore().Publish(&Event{Name: eventName, Data: data})
}

// PublishEventSync method publishes events to subscribed callbacks synchronously.
func PublishEventSync(eventName string, data interface{}) {
	AppEventStore().PublishSync(&Event{Name: eventName, Data: data})
}

// SubscribeEvent method is to subscribe to new or existing event.
func SubscribeEvent(eventName string, ec EventCallback) {
	AppEventStore().Subscribe(eventName, ec)
}

// UnsubscribeEvent method is to unsubscribe by event name and `EventCallback`
// from app event store.
func UnsubscribeEvent(eventName string, ec EventCallback) {
	UnsubscribeEventf(eventName, ec.Callback)
}

// UnsubscribeEventf method is to unsubscribe by event name and `EventCallbackFunc`
// from app event store.
func UnsubscribeEventf(eventName string, ec EventCallbackFunc) {
	AppEventStore().Unsubscribe(eventName, ec)
}

// OnInit method is to subscribe to aah application `OnInit` event. `OnInit` event
// published right after the aah application configuration `aah.conf` initialized.
func OnInit(ecb EventCallbackFunc, priority ...int) {
	AppEventStore().Subscribe(EventOnInit, EventCallback{
		Callback:    ecb,
		PublishOnce: true,
		priority:    parsePriority(priority...),
	})
}

// OnStart method is to subscribe to aah application `OnStart` event. `OnStart`
// event pubished right before the aah server listen and serving request.
func OnStart(ecb EventCallbackFunc, priority ...int) {
	AppEventStore().Subscribe(EventOnStart, EventCallback{
		Callback:    ecb,
		PublishOnce: true,
		priority:    parsePriority(priority...),
	})
}

// OnRequest method is to subscribe to aah server `OnRequest` extension point.
// `OnRequest` called for every incoming request.
//
// The `aah.Context` object passed to the extension functions is decorated with
// the `ctx.SetURL()` and `ctx.SetMethod()` methods. Calls to these methods will
// impact how the request is routed and can be used for rewrite rules.
//
// Route is not yet processed at this point.
func OnRequest(sef EventCallbackFunc) {
	if onRequestFunc == nil {
		onRequestFunc = sef
		return
	}
	log.Warn("'OnRequest' aah server extension function is already subscribed.")
}

// OnPreReply method is to subscribe to aah server `OnPreReply` extension point.
// `OnPreReply` called for every reply from aah server.
//
// Except when 1) Static file request, 2) `Reply().Done()`
// 3) `Reply().Redirect(...)` is called. Refer `aah.Reply.Done()` godoc for more info.
func OnPreReply(sef EventCallbackFunc) {
	if onPreReplyFunc == nil {
		onPreReplyFunc = sef
		return
	}
	log.Warn("'OnPreReply' aah server extension function is already subscribed.")
}

// OnAfterReply method is to subscribe to aah server `OnAfterReply` extension point.
// `OnAfterReply` called for every reply from aah server.
//
// Except when 1) Static file request, 2) `Reply().Done()`
// 3) `Reply().Redirect(...)` is called. Refer `aah.Reply.Done()` godoc for more info.
func OnAfterReply(sef EventCallbackFunc) {
	if onAfterReplyFunc == nil {
		onAfterReplyFunc = sef
		return
	}
	log.Warn("'OnAfterReply' aah server extension function is already subscribed.")
}

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// EventStore methods
//___________________________________

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

	log.Debugf("Event [%s] published in asynchronous mode", e.Name)
	for idx, ec := range es.subscribers[e.Name] {
		if ec.PublishOnce {
			if !ec.published {
				go func(event *Event, ecb EventCallbackFunc) {
					ecb(event)
				}(e, ec.Callback)

				es.subscribers[e.Name][idx].published = true
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

	log.Debugf("Event [%s] publishing in synchronous mode", e.Name)
	for idx, ec := range es.subscribers[e.Name] {
		if ec.PublishOnce {
			if !ec.published {
				ec.Callback(e)
				es.subscribers[e.Name][idx].published = true
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
		log.Debugf("Callback: %s, subscribed to event: %s", funcName(ec.Callback), event)
		return
	}

	es.subscribers[event] = EventCallbacks{}
	es.subscribers[event] = append(es.subscribers[event], ec)
	log.Debugf("Callback: %s, subscribed to event: %s", funcName(ec.Callback), event)
}

// Unsubscribe method is to unsubscribe any callback from event store by event.
func (es *EventStore) Unsubscribe(event string, callback EventCallbackFunc) {
	es.mu.Lock()
	defer es.mu.Unlock()
	if !es.IsEventExists(event) {
		log.Warnf("Subscribers not exists for event: %s", event)
		return
	}

	for idx := len(es.subscribers[event]) - 1; idx >= 0; idx-- {
		ec := es.subscribers[event][idx]
		if funcEqual(ec.Callback, callback) {
			es.subscribers[event] = append(es.subscribers[event][:idx], es.subscribers[event][idx+1:]...)
			log.Debugf("Callback: %s, unsubscribed from event: %s", funcName(callback), event)
			return
		}
	}

	log.Warnf("Given callback: %s, not found in eventStore for event: %s", funcName(callback), event)
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

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// EventCallbacks methods
//___________________________________

// Sort interface for EventCallbacks
func (ec EventCallbacks) Len() int           { return len(ec) }
func (ec EventCallbacks) Less(i, j int) bool { return ec[i].priority < ec[j].priority }
func (ec EventCallbacks) Swap(i, j int)      { ec[i], ec[j] = ec[j], ec[i] }

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// Unexported methods
//___________________________________

func publishOnRequestEvent(ctx *Context) {
	if onRequestFunc != nil {
		ctx.decorated = true
		onRequestFunc(&Event{Name: EventOnRequest, Data: ctx})
		ctx.decorated = false
	}
}

func publishOnPreReplyEvent(ctx *Context) {
	if onPreReplyFunc != nil {
		onPreReplyFunc(&Event{Name: EventOnPreReply, Data: ctx})
	}
}

func publishOnAfterReplyEvent(ctx *Context) {
	if onAfterReplyFunc != nil {
		onAfterReplyFunc(&Event{Name: EventOnAfterReply, Data: ctx})
	}
}

// funcEqual method to compare to function callback interface data. In effect
// comparing the pointers of the indirect layer. Read more about the
// representation of functions here: http://golang.org/s/go11func
func funcEqual(a, b interface{}) bool {
	av := reflect.ValueOf(&a).Elem()
	bv := reflect.ValueOf(&b).Elem()
	return av.InterfaceData() == bv.InterfaceData()
}

// funcName method to get callback function name.
func funcName(f interface{}) string {
	fi := ess.GetFunctionInfo(f)
	return fi.Name
}

func parsePriority(priority ...int) int {
	pr := 1 // default priority is 1
	if len(priority) > 0 && priority[0] > 0 {
		pr = priority[0]
	}
	return pr
}
