// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"aahframework.org/test.v0/assert"
)

func TestOnInitEvent(t *testing.T) {
	onInitFunc1 := func(e *Event) {
		fmt.Println("onInitFunc1:", e)
	}

	onInitFunc2 := func(e *Event) {
		fmt.Println("onInitFunc2:", e)
	}

	onInitFunc3 := func(e *Event) {
		fmt.Println("onInitFunc3:", e)
	}

	appEventStore = &EventStore{subscribers: make(map[string]EventCallbacks), mu: &sync.Mutex{}}
	assert.False(t, AppEventStore().IsEventExists(EventOnInit))

	OnInit(onInitFunc1)
	assert.True(t, AppEventStore().IsEventExists(EventOnInit))
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnInit))

	OnInit(onInitFunc3, 4)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnInit))

	OnInit(onInitFunc2, 2)
	assert.Equal(t, 3, AppEventStore().SubscriberCount(EventOnInit))

	// publish 1
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnInit, Data: "On Init event published 1"})

	AppEventStore().Unsubscribe(EventOnInit, onInitFunc2)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnInit))

	// publish 2
	PublishEventSync(EventOnInit, "On Init event published 2")

	AppEventStore().Unsubscribe(EventOnInit, onInitFunc1)
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnInit))

	// publish 3
	PublishEventSync(EventOnInit, "On Init event published 3")
	PublishEventSync(EventOnStart, "On start not gonna fire")

	// event not exists
	AppEventStore().Unsubscribe(EventOnStart, onInitFunc1)

	AppEventStore().Unsubscribe(EventOnInit, onInitFunc3)
	assert.Equal(t, 0, AppEventStore().SubscriberCount(EventOnInit))
	assert.Equal(t, 0, AppEventStore().SubscriberCount(EventOnStart))

	// EventOnInit not exists
	AppEventStore().Unsubscribe(EventOnInit, onInitFunc3)
}

func TestOnStartEvent(t *testing.T) {
	onStartFunc1 := func(e *Event) {
		fmt.Println("onStartFunc1:", e)
	}

	onStartFunc2 := func(e *Event) {
		fmt.Println("onStartFunc2:", e)
	}

	onStartFunc3 := func(e *Event) {
		fmt.Println("onStartFunc3:", e)
	}

	appEventStore = &EventStore{subscribers: make(map[string]EventCallbacks), mu: &sync.Mutex{}}
	assert.False(t, AppEventStore().IsEventExists(EventOnStart))

	OnStart(onStartFunc1)
	assert.True(t, AppEventStore().IsEventExists(EventOnStart))
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnStart))

	OnStart(onStartFunc3, 4)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnStart))

	OnStart(onStartFunc2, 2)
	assert.Equal(t, 3, AppEventStore().SubscriberCount(EventOnStart))

	// publish 1
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnStart, Data: "On start event published 1"})

	AppEventStore().Unsubscribe(EventOnStart, onStartFunc2)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnStart))

	// publish 2
	PublishEventSync(EventOnStart, "On start event published 2")

	AppEventStore().Unsubscribe(EventOnStart, onStartFunc1)
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnStart))

	// publish 3
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnStart, Data: "On start event published 3"})
	PublishEventSync(EventOnInit, "On init not gonna fire")

	// event not exists
	AppEventStore().Unsubscribe(EventOnInit, onStartFunc1)

	AppEventStore().Unsubscribe(EventOnStart, onStartFunc3)
	assert.Equal(t, 0, AppEventStore().SubscriberCount(EventOnStart))
	assert.Equal(t, 0, AppEventStore().SubscriberCount(EventOnInit))

	// EventOnInit not exists
	AppEventStore().Unsubscribe(EventOnStart, onStartFunc3)
}

func TestOnShutdownEvent(t *testing.T) {
	onShutdownFunc1 := func(e *Event) {
		fmt.Println("onShutdownFunc1:", e)
	}

	onShutdownFunc2 := func(e *Event) {
		fmt.Println("onShutdownFunc2:", e)
	}

	onShutdownFunc3 := func(e *Event) {
		fmt.Println("onShutdownFunc3:", e)
	}

	appEventStore = &EventStore{subscribers: make(map[string]EventCallbacks), mu: &sync.Mutex{}}
	assert.False(t, AppEventStore().IsEventExists(EventOnShutdown))

	OnShutdown(onShutdownFunc1)
	assert.True(t, AppEventStore().IsEventExists(EventOnShutdown))
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnShutdown))

	OnShutdown(onShutdownFunc3, 4)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnShutdown))

	OnShutdown(onShutdownFunc2, 2)
	assert.Equal(t, 3, AppEventStore().SubscriberCount(EventOnShutdown))

	// publish 1
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown, Data: "On shutdown event published 1"})

	AppEventStore().Unsubscribe(EventOnShutdown, onShutdownFunc2)
	assert.Equal(t, 2, AppEventStore().SubscriberCount(EventOnShutdown))

	// publish 2
	PublishEventSync(EventOnShutdown, "On shutdown event published 2")

	AppEventStore().Unsubscribe(EventOnShutdown, onShutdownFunc1)
	assert.Equal(t, 1, AppEventStore().SubscriberCount(EventOnShutdown))

	// publish 3
	AppEventStore().sortAndPublishSync(&Event{Name: EventOnShutdown, Data: "On shutdown event published 3"})
	PublishEventSync(EventOnShutdown, "On shutdown not gonna fire")

	// event not exists
	AppEventStore().Unsubscribe(EventOnShutdown, onShutdownFunc1)

	AppEventStore().Unsubscribe(EventOnShutdown, onShutdownFunc3)
	assert.Equal(t, 0, AppEventStore().SubscriberCount(EventOnShutdown))

	// EventOnShutdown not exists
	AppEventStore().Unsubscribe(EventOnShutdown, onShutdownFunc3)
}

func TestServerExtensionEvent(t *testing.T) {
	// OnRequest
	assert.Nil(t, onRequestFunc)
	publishOnRequestEvent(&Context{})
	OnRequest(func(e *Event) {
		t.Log("OnRequest event func called")
	})
	assert.NotNil(t, onRequestFunc)

	onRequestFunc(&Event{Name: EventOnRequest, Data: "request Data OnRequest"})
	publishOnRequestEvent(&Context{})
	OnRequest(func(e *Event) {
		t.Log("OnRequest event func called 2")
	})

	// OnPreReply
	assert.Nil(t, onPreReplyFunc)
	publishOnPreReplyEvent(&Context{})
	OnPreReply(func(e *Event) {
		t.Log("OnPreReply event func called")
	})
	assert.NotNil(t, onPreReplyFunc)

	onPreReplyFunc(&Event{Name: EventOnPreReply, Data: "Context Data OnPreReply"})
	publishOnPreReplyEvent(&Context{})
	OnPreReply(func(e *Event) {
		t.Log("OnPreReply event func called 2")
	})

	// OnAfterReply
	assert.Nil(t, onAfterReplyFunc)
	publishOnAfterReplyEvent(&Context{})
	OnAfterReply(func(e *Event) {
		t.Log("OnAfterReply event func called")
	})
	assert.NotNil(t, onAfterReplyFunc)

	onAfterReplyFunc(&Event{Name: EventOnAfterReply, Data: "Context Data OnAfterReply"})
	publishOnAfterReplyEvent(&Context{})
	OnAfterReply(func(e *Event) {
		t.Log("OnAfterReply event func called 2")
	})
}

func TestSubscribeAndUnsubscribeAndPublish(t *testing.T) {
	myEventFunc1 := func(e *Event) {
		fmt.Println("myEventFunc1:", e)
	}

	myEventFunc2 := func(e *Event) {
		fmt.Println("myEventFunc2:", e)
	}

	myEventFunc3 := func(e *Event) {
		fmt.Println("myEventFunc3:", e)
	}

	ecb1 := EventCallback{Callback: myEventFunc1, CallOnce: true}
	assert.Equal(t, 0, AppEventStore().SubscriberCount("myEvent1"))
	SubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 1, AppEventStore().SubscriberCount("myEvent1"))

	SubscribeEvent("myEvent1", EventCallback{Callback: myEventFunc2})
	SubscribeEventf("myEvent1", myEventFunc2)
	assert.Equal(t, 3, AppEventStore().SubscriberCount("myEvent1"))

	assert.Equal(t, 0, AppEventStore().SubscriberCount("myEvent2"))
	SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3})
	assert.Equal(t, 1, AppEventStore().SubscriberCount("myEvent2"))

	PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	UnsubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 2, AppEventStore().SubscriberCount("myEvent1"))

	PublishEvent("myEvent1", "myEvent1 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	PublishEvent("myEventNotExists", nil)

	SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3, CallOnce: true})
	PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	PublishEventSync("myEvent2", "myEvent2 is fired sync")
}
