// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/aah source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package aah

import (
	"path/filepath"
	"testing"
	"time"

	"aahframework.org/test.v0/assert"
)

func TestEvenOnInit(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Event Publisher]: %s", ts.URL)

	// declare functions
	onInitFunc1 := func(e *Event) {
		t.Log("onInitFunc1:", e)
	}

	onInitFunc2 := func(e *Event) {
		t.Log("onInitFunc2:", e)
	}

	onInitFunc3 := func(e *Event) {
		t.Log("onInitFunc3:", e)
	}

	es := ts.app.eventStore
	assert.False(t, es.IsEventExists(EventOnInit))

	ts.app.OnInit(onInitFunc1)
	assert.True(t, es.IsEventExists(EventOnInit))
	assert.Equal(t, 1, es.SubscriberCount(EventOnInit))

	ts.app.OnInit(onInitFunc3, 4)
	assert.Equal(t, 2, es.SubscriberCount(EventOnInit))

	ts.app.OnInit(onInitFunc2, 2)
	assert.Equal(t, 3, es.SubscriberCount(EventOnInit))

	// publish 1
	es.sortAndPublishSync(&Event{Name: EventOnInit, Data: "On Init event published 1"})

	es.Unsubscribe(EventOnInit, onInitFunc2)
	assert.Equal(t, 2, es.SubscriberCount(EventOnInit))

	// publish 2
	ts.app.PublishEventSync(EventOnInit, "On Init event published 2")

	es.Unsubscribe(EventOnInit, onInitFunc1)
	assert.Equal(t, 1, es.SubscriberCount(EventOnInit))

	// publish 3
	ts.app.PublishEventSync(EventOnInit, "On Init event published 3")
	ts.app.PublishEventSync(EventOnStart, "On start not gonna fire")

	// event not exists
	es.Unsubscribe(EventOnStart, onInitFunc1)

	es.Unsubscribe(EventOnInit, onInitFunc3)
	assert.Equal(t, 0, es.SubscriberCount(EventOnInit))
	assert.Equal(t, 0, es.SubscriberCount(EventOnStart))

	// EventOnInit not exists
	es.Unsubscribe(EventOnInit, onInitFunc3)
}

func TestEventOnStart(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Event Publisher]: %s", ts.URL)

	// declare functions
	onStartFunc1 := func(e *Event) {
		t.Log("onStartFunc1:", e)
	}

	onStartFunc2 := func(e *Event) {
		t.Log("onStartFunc2:", e)
	}

	onStartFunc3 := func(e *Event) {
		t.Log("onStartFunc3:", e)
	}

	es := ts.app.eventStore
	assert.False(t, es.IsEventExists(EventOnStart))

	ts.app.OnStart(onStartFunc1)
	assert.True(t, es.IsEventExists(EventOnStart))
	assert.Equal(t, 1, es.SubscriberCount(EventOnStart))

	ts.app.OnStart(onStartFunc3, 4)
	assert.Equal(t, 2, es.SubscriberCount(EventOnStart))

	ts.app.OnStart(onStartFunc2, 2)
	assert.Equal(t, 3, es.SubscriberCount(EventOnStart))

	// publish 1
	es.sortAndPublishSync(&Event{Name: EventOnStart, Data: "On start event published 1"})

	es.Unsubscribe(EventOnStart, onStartFunc2)
	assert.Equal(t, 2, es.SubscriberCount(EventOnStart))

	// publish 2
	ts.app.PublishEventSync(EventOnStart, "On start event published 2")

	es.Unsubscribe(EventOnStart, onStartFunc1)
	assert.Equal(t, 1, es.SubscriberCount(EventOnStart))

	// publish 3
	es.sortAndPublishSync(&Event{Name: EventOnStart, Data: "On start event published 3"})
	ts.app.PublishEventSync(EventOnInit, "On init not gonna fire")

	// event not exists
	es.Unsubscribe(EventOnInit, onStartFunc1)

	es.Unsubscribe(EventOnStart, onStartFunc3)
	assert.Equal(t, 0, es.SubscriberCount(EventOnStart))
	assert.Equal(t, 0, es.SubscriberCount(EventOnInit))

	// EventOnInit not exists
	es.Unsubscribe(EventOnStart, onStartFunc3)
}

func TestEventOnShutdown(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Event Publisher]: %s", ts.URL)

	// declare functions
	onShutdownFunc1 := func(e *Event) {
		t.Log("onShutdownFunc1:", e)
	}

	onShutdownFunc2 := func(e *Event) {
		t.Log("onShutdownFunc2:", e)
	}

	onShutdownFunc3 := func(e *Event) {
		t.Log("onShutdownFunc3:", e)
	}

	es := ts.app.eventStore
	assert.False(t, es.IsEventExists(EventOnShutdown))

	ts.app.OnShutdown(onShutdownFunc1)
	assert.True(t, es.IsEventExists(EventOnShutdown))
	assert.Equal(t, 1, es.SubscriberCount(EventOnShutdown))

	ts.app.OnShutdown(onShutdownFunc3, 4)
	assert.Equal(t, 2, es.SubscriberCount(EventOnShutdown))

	ts.app.OnShutdown(onShutdownFunc2, 2)
	assert.Equal(t, 3, es.SubscriberCount(EventOnShutdown))

	// publish 1
	es.sortAndPublishSync(&Event{Name: EventOnShutdown, Data: "On shutdown event published 1"})

	es.Unsubscribe(EventOnShutdown, onShutdownFunc2)
	assert.Equal(t, 2, es.SubscriberCount(EventOnShutdown))

	// publish 2
	ts.app.PublishEventSync(EventOnShutdown, "On shutdown event published 2")

	es.Unsubscribe(EventOnShutdown, onShutdownFunc1)
	assert.Equal(t, 1, es.SubscriberCount(EventOnShutdown))

	// publish 3
	es.sortAndPublishSync(&Event{Name: EventOnShutdown, Data: "On shutdown event published 3"})
	ts.app.PublishEventSync(EventOnShutdown, "On shutdown not gonna fire")

	// event not exists
	es.Unsubscribe(EventOnShutdown, onShutdownFunc1)

	es.Unsubscribe(EventOnShutdown, onShutdownFunc3)
	assert.Equal(t, 0, es.SubscriberCount(EventOnShutdown))

	// EventOnShutdown not exists
	es.Unsubscribe(EventOnShutdown, onShutdownFunc3)
}
func TestEventSubscribeAndUnsubscribeAndPublish(t *testing.T) {
	importPath := filepath.Join(testdataBaseDir(), "webapp1")
	ts, err := newTestServer(t, importPath)
	assert.Nil(t, err)
	defer ts.Close()

	t.Logf("Test Server URL [Event Publisher]: %s", ts.URL)

	// declare functions
	myEventFunc1 := func(e *Event) {
		t.Log("myEventFunc1:", e)
	}

	myEventFunc2 := func(e *Event) {
		t.Log("myEventFunc2:", e)
	}

	myEventFunc3 := func(e *Event) {
		t.Log("myEventFunc3:", e)
	}

	es := ts.app.eventStore

	ecb1 := EventCallback{Callback: myEventFunc1, CallOnce: true}
	assert.Equal(t, 0, es.SubscriberCount("myEvent1"))
	ts.app.SubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 1, es.SubscriberCount("myEvent1"))

	ts.app.SubscribeEvent("myEvent1", EventCallback{Callback: myEventFunc2})
	ts.app.SubscribeEventf("myEvent1", myEventFunc2)
	assert.Equal(t, 3, es.SubscriberCount("myEvent1"))

	assert.Equal(t, 0, es.SubscriberCount("myEvent2"))
	ts.app.SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3})
	assert.Equal(t, 1, es.SubscriberCount("myEvent2"))

	ts.app.PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.UnsubscribeEvent("myEvent1", ecb1)
	assert.Equal(t, 2, es.SubscriberCount("myEvent1"))

	ts.app.PublishEvent("myEvent1", "myEvent1 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.PublishEvent("myEventNotExists", nil)

	ts.app.SubscribeEvent("myEvent2", EventCallback{Callback: myEventFunc3, CallOnce: true})
	ts.app.PublishEvent("myEvent2", "myEvent2 is fired async")
	time.Sleep(time.Millisecond * 100) // for goroutine to finish

	ts.app.PublishEventSync("myEvent2", "myEvent2 is fired sync")
}
